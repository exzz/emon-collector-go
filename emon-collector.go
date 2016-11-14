package main

import (
	"flag"
	"log"
	"time"

	toml "github.com/BurntSushi/toml"
	emon "github.com/exzz/emon-api-go"
	client "github.com/influxdata/influxdb/client/v2"
)

// Command line flag
var fConfig = flag.String("config", "", "configuration file to load")
var fVerbose = flag.Bool("verbose", false, "log read value")

// Configuration file
type collectorConfig struct {
	Emon     emonConfig
	Influxdb influxdbConfig
}

type emonConfig struct {
	URL           string
	FetchInterval int
}

type influxdbConfig struct {
	URL             string
	Username        string
	Password        string
	Database        string
	RetentionPolicy string
	Precision       string
}

var config collectorConfig

func main() {

	// Parse command line flags
	flag.Parse()

	if _, err := toml.DecodeFile(*fConfig, &config); err != nil {
		log.Fatalf("Cannot parse config file: %s", err)
	}

	log.Printf("Starting %v\n", config)

	// set up
	emon, err := emon.NewClient(emon.Config{
		URL: config.Emon.URL,
	})
	if err != nil {
		log.Fatalf("Unable to connect to Energymonitor: %s", err)
	}

	influxdb, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     config.Influxdb.URL,
		Username: config.Influxdb.Username,
		Password: config.Influxdb.Password,
	})
	if err != nil {
		log.Fatalf("Unable to connect to InfluxDB: %s", err.Error())
	}

	// main loop
	tickerWrite := time.NewTicker(time.Duration(config.Emon.FetchInterval) * time.Second)

	for {

		select {
		case <-tickerWrite.C:

			// read data
			err := emon.Read()
			if err != nil {
				log.Printf("Cannot fetch EnergyMonitor data: %s", err)
				continue
			}

			// create influxdb point
			bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
				Database:        config.Influxdb.Database,
				Precision:       config.Influxdb.Precision,
				RetentionPolicy: config.Influxdb.RetentionPolicy,
			})

			tags := map[string]string{"site": emon.Sensor.SiteName, "module": emon.Sensor.ModuleName}
			fields := map[string]interface{}{
				"real":     emon.Sensor.RealPower,
				"apparent": emon.Sensor.ApparentPower,
			}

			pt, err := client.NewPoint("power", tags, fields, time.Unix(int64(emon.Sensor.Time), 0))
			if err != nil {
				log.Printf("Cannot create infludb point: %s", err.Error())
				continue
			}
			bp.AddPoint(pt)

			// write infludb point
			err = influxdb.Write(bp)
			if err != nil {
				log.Printf("Cannot write infludb point: %s", err.Error())
				continue
			}

			// log read value
			if *fVerbose {
				log.Printf("%+v\n", emon.Sensor)
			}
		}
	}

}
