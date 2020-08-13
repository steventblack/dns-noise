# dns-noise
This is a simple utility designed to work in conjunction with a local Pihole to generate a steady stream of DNS queries against random domains.
The random DNS queries provide a degree of noise that makes discerning user-driven DNS queries more difficult to identify. 
The default list of random domains is pulled from [Cisco Umbrella](https://umbrella.cisco.com/blog/cisco-umbrella-1-million), 
which provide a list of the top 1,000,000 accessed domains and is updated every 24h.

## General Information ##
This service can utilize a pihole to dynamically adjust its own query rate. If configured properly, it polls the pihole's query activity and adjusts
its own rate based on the activity level returned. If a pihole is not available (or not configured properly), it will generate a random rate value between 
a stated min/max interval. A new random rate will be generated periodically. 

The noise generated from this service can obfuscate typical attempts to identify or track user activity based on domain lookups. However,
a determined party may still be able to differentiate the noise from legitimate traffic given enough time, activity logs, and effort.

## Running ##
```dns-noise [-c|--conf confpath] [-d|--database dbpath] [-r|--reusedb] --min min_interval --max max_interval
-c|--conf confpath
  Specifies the path to the configuration file. 
  Default path is "dns-noise.conf".
-d|--databse dbpath
  Specifies the path for the database with the list of "noise" domains. 
  Default path is "/tmp/dns-noise.db"
-r|--reusedb
  Boolean flag used to prevent refreshing the "noise" domains database on startup. 
  Default is false.
--min min_interval
  Specifies the minimum duration between queries. 
  It accepts any duration string that can be parsed by Go's time.ParseDuration. Default is 100ms.
--max max_interval
  Specifies the maximum duration between queries. 
  It accepts any duration string that can be parsed by Go's time.ParseDuration. Default is 15s.
```

## Installation ##
_Coming Soon_

## Config ##
_Coming Soon_
