# UPS CH Aggregator
A simple service that will retrieve UPS load information and dump it into a simple ClickHouse table.

## Supported UPSes

This aggregator currently supports any CyberPower UPS that is compatible with CyberPower's Power Panel Management software. 

I've done some light digging into their APIs as well as how the authentication works and it is fairly straightforward except the frontend does some sort of AES encryption on the username and password before (???) sending it to the backend (during auth) which then returns a tiny JWT. 

The key needed to generate the hashes for login is probably buried somewhere in that mess of an application, I am hoping when I have some more free time to find it so we can bypass needing to look at the browser dev tools network tab to steal the hashes.

Default login is `admin`/`admin` and the hashes are `2E04DF62D2DD379F7F95BE8EC627C7CB`/`2E04DF62D2DD379F7F95BE8EC627C7CB`

### Validated Models
- OR1500PFCRT2U ([CyberPower Website](https://www.cyberpowersystems.com/product/ups/pfc-sinewave/or1500pfcrt2u/))

## Configuration

| Environment Variable                             | Default                             | Description                                                           |
|--------------------------------------------------|-------------------------------------|-----------------------------------------------------------------------|
| `UPS_CH_CLICK_HOUSE_ADDRESSES`                   | `172.16.11.107:19000`               | The address(es) of your ClickHouse server(s).                         |
| `UPS_CH_CLICK_HOUSE_DATABASE`                    | `ups_aggregator`                    | The ClickHouse database to use.                                       |
| `UPS_CH_CLICK_HOUSE_USERNAME`                    | `default`                           | The username to connect with.                                         |
| `UPS_CH_CLICK_HOUSE_PASSWORD`                    | `n/a`                               | The password to use for connecting.                                   |
| `UPS_CH_CYBER_POWER_POWER_PANEL_URL`             | `http://10.0.0.250:3052/management` | The URL to your Power Panel Management instance.                      |
| `UPS_CH_CYBER_POWER_POWER_PANEL_HASHED_USERNAME` | `2E04DF62D2DD379F7F95BE8EC627C7CB`  | The hashed username that the frontend passes to the backend for auth. |
| `UPS_CH_CYBER_POWER_POWER_PANEL_HASHED_PASSWORD` | `2E04DF62D2DD379F7F95BE8EC627C7CB`  | The hashed password that the frontend passes to the backend for auth. |

### Grafana

Here is a simple query to get the time series data out of ClickHouse, and render a simple chart.

```SQL
SELECT $__timeInterval(Timestamp) as time, DisplayName, max(Watts) FROM power_readings WHERE $__timeFilter(Timestamp) GROUP BY DisplayName, time ORDER BY time ASC LIMIT 1000
```