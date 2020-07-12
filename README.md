# dns-noise
This is a simple utility designed to work in conjunction with a local Pihole to generate a steady stream of DNS queries against random domains.
The random DNS queries provide a degree of noise that makes discerning user-driven DNS queries more difficult to identify. 
The list of random domains is pulled from [Cisco Umbrella](https://umbrella.cisco.com/blog/cisco-umbrella-1-million), 
which provide a list of the top 1,000,000 accessed domains and is updated every 24h. 

## General Information ##
This service utilizes the query activity of the local Pihole to dynamically adjust its own query rate. It polls the Pihole every minute
using the Pihole API, subtracts out the queries the dns-noise service initiated, and generates a 5-minute moving average of the query rate.
The dns-noise service uses that information to limit its own activity to ~10% of the moving rate average by adjusting its sleep interval between calls.
In order to provide a bit more randomness, the sleep value itself is modified by a random 0-10% on each iteration. Upon each iteration, a random
domain is selected from the list and DNS query for it is issued.

The noise generated from this service should provide reasonable protection against any typical attempts to identify or track user activity. However,
a determined party may still be able to differentiate the noise from legitimate traffic given enough time, activity logs, and effort.

## Installation ##
_Coming Soon_

## Config ##
_Coming Soon_
