package config

type UPSAggregatorConfig struct {
	ClickHouseAddresses []string `split_words:"true" required:"true" default:"172.16.11.107:19000"`
	ClickHouseDatabase  string   `split_words:"true" required:"true" default:"ups_aggregator"`
	ClickHouseUsername  string   `split_words:"true" required:"true" default:"default"`
	ClickHousePassword  string   `split_words:"true"`

	CyberPowerPowerPanelURL            string `split_words:"true" required:"true" default:"http://10.0.0.250:3052/management"`
	CyberPowerPowerPanelHashedUsername string `split_words:"true" required:"true" default:"2E04DF62D2DD379F7F95BE8EC627C7CB"`
	CyberPowerPowerPanelHashedPassword string `split_words:"true" required:"true" default:"2E04DF62D2DD379F7F95BE8EC627C7CB"`
}
