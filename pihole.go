//
// Copyright 2020 Steven T Black
//

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// Example response from Pihole
// It's not particularly well-structured JSON, but it'll do for this purpose
// {"data":[["1593882001","AAAA","trk.pinterest.com","impala.local","1","0","4","8","N\/A","-1"],["1593882001","A","trk.pinterest.com","impala.local","1","0","4","7","N\/A","-1"]]}
type PiholeQueries struct {
	Data [][]string
}

// piholeFetchActivity polls the configured pihole for query activity.
// It accepts the pihole configuration information block and returns the number of queries observed.
// On error, it returns a value of 0.
func piholeFetchActivity(p *Pihole) (int, error) {
	until := time.Now().Unix()
	from := until - int64(p.ActivityPeriod.Duration().Seconds())

	// Time values need to be expressed in Unix epoch time format
	url := fmt.Sprintf("http://%s/admin/api.php?getAllQueries&from=%d&until=%d&auth=%s", p.Host, from, until, p.AuthToken)

	response, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("Unable to fetch activity data from '%s'; status '%s'", p.Host, response.Status)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Unexpected status  from '%s'; status '%s'", p.Host, response.Status)
	}

	jsonBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return 0, err
	}

	var queries PiholeQueries
	err = json.Unmarshal(jsonBody, &queries)
	if err != nil {
		return 0, err
	}

	// Filters out entries from dns-noise host (if applicable)
	numQueries := piholeFilterNoise(p.Filter, queries.Data)
	if numQueries <= 0 {
		return 0, fmt.Errorf("No activity available from pihole")
	}

	return numQueries, nil
}

// piholeFilterNoise removes the queries from the filtered host from the query activity total.
// If the filter string is empty, then it simply returns the number of queries in the set.
// It returns the adjusted total number of queries in the set.
func piholeFilterNoise(filter string, queries [][]string) int {
	if filter == "" {
		return len(queries)
	}

	var numQueries int
	for _, query := range queries {
		if !strings.HasPrefix(query[3], filter) {
			numQueries++
		}
	}

	return numQueries
}

// piholeEnabled checks the necessary settings are present in the config for pihole utilization.
// It does not perform any validation checks on the setting values.
// It returns a bool reflecting the configuration is setup or not.
func piholeEnabled(p *Pihole) bool {
	enabled := true

	if p.Host == "" {
		enabled = false
	}
	if p.AuthToken == "" {
		enabled = false
	}
	if p.NoisePercentage <= 0 {
		enabled = false
	}

	return enabled
}
