package main

import (
	"context"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/kelseyhightower/envconfig"
	"github.com/slickwilli/ups-ch/config"
	"github.com/slickwilli/ups-ch/models"
	"github.com/slickwilli/ups-ch/pkg/clients/cyberpower"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logConf := zap.NewProductionConfig()
	logConf.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	logConf.DisableCaller = true
	logger, err := logConf.Build()
	if err != nil {
		log.Fatal("error building zap logger", err)
	}

	var conf config.UPSAggregatorConfig
	if err := envconfig.Process("UPS_CH", &conf); err != nil {
		logger.Fatal("unable to build configuration", zap.Error(err))
	}

	agg, err := NewAggregator(logger, &conf)
	if err != nil {
		logger.Fatal("unable to build aggregator", zap.Error(err))
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)

	go agg.StartAggregator(30*time.Second, done)

	<-sigs
	logger.Info("exiting ups ch aggregator")
	close(done)
}

type Aggregator struct {
	conn             clickhouse.Conn
	powerPanelClient *cyberpower.Client
	logger           *zap.Logger
}

func NewAggregator(logger *zap.Logger, conf *config.UPSAggregatorConfig) (*Aggregator, error) {
	logger = logger.Named("aggregator")
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: conf.ClickHouseAddresses,
		Auth: clickhouse.Auth{
			Database: conf.ClickHouseDatabase,
			Username: conf.ClickHouseUsername,
			Password: conf.ClickHousePassword,
		},
	})
	if err != nil {
		return nil, err
	}
	v, err := conn.ServerVersion()
	if err != nil {
		return nil, err
	}
	logger.Info("connected to clickhouse server", zap.String("version", v.Version.String()), zap.Uint64("revision", v.Revision))
	if err := conn.Exec(context.Background(), `
CREATE TABLE IF NOT EXISTS ups_aggregator.power_readings (
	DeviceID UInt32,
	DisplayName String,
	Watts Float32,
	Timestamp DateTime
)
ENGINE = MergeTree
PRIMARY KEY (DeviceID, Timestamp)
`); err != nil {
		return nil, err
	}
	cyberPowerClient, err := cyberpower.NewClient(
		conf.CyberPowerPowerPanelURL,
		conf.CyberPowerPowerPanelHashedUsername,
		conf.CyberPowerPowerPanelHashedPassword,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &Aggregator{
		conn:             conn,
		powerPanelClient: cyberPowerClient,
		logger:           logger,
	}, nil
}

func (a *Aggregator) StartAggregator(interval time.Duration, done chan bool) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-done:
			a.logger.Info("stopping aggregator")
			return
		case t := <-ticker.C:
			managementTree, err := a.powerPanelClient.GetManagementTree()
			if err != nil {
				a.logger.Error("error getting CyberPower PowerPanel management tree", zap.Error(err))
				continue
			}
			batch, err := a.conn.PrepareBatch(context.Background(), "INSERT INTO power_readings")
			if err != nil {
				a.logger.Error("error preparing clickhouse batch insert", zap.Error(err))
				continue
			}
			for _, n := range managementTree.ChildrenNodeList {
				if n.Type == cyberpower.ManagementNodeTypeUPS {
					a.logger.Info(
						"found ups",
						zap.Int("id", n.ID),
						zap.String("name", n.Name),
						zap.Float32("current_load_watts", n.NodeBrief.OutputLoad.CurrentWatts),
						zap.Int("current_load_percentage", n.NodeBrief.OutputLoad.Percentage),
					)
					if err := batch.AppendStruct(&models.Reading{
						DeviceID:    uint32(n.ID),
						DisplayName: n.Name,
						Watts:       n.NodeBrief.OutputLoad.CurrentWatts,
						Timestamp:   t,
					}); err != nil {
						a.logger.Warn("error appending reading to clickhouse batch")
						continue
					}
				}
			}
			if err := batch.Send(); err != nil {
				a.logger.Error("error sending clickhouse batch insert", zap.Error(err))
				continue
			}
		}
	}
}
