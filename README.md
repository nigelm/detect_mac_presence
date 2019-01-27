# Detect Presence By MAC Address

This is a go program to detect the presense or absence of people at home by
detecting whether the MAC address of their phone is known to the local router
(in my case a `pfsense` firewall).

This is to work with the Presence handling for SmartThings documented by
@fuzzsb in
https://github.com/fuzzysb/SmartThings/tree/master/smartapps/fuzzysb/asuswrt-wifi-presence.src
and https://community.smartthings.com/t/release-asuswrt-wifi-presence/37802

This is a slightly more sophisticated mechanism on the router end than his
shell script. The implementation in go was purely to give me a push to try
coding this in go (and I can easily cross compile it to move the binary on to
my pfsense box).

## Summary of Operation

When run the code first loads its state file from a file -
normally `~/.presence.json`

It then runs and examines each line of output from `arp -an`

For each defined user in the config, the MAC address is searched out using a
really basic regexp (exact match - which means you could finesse this by using
a alternate regexp).

If a user has changed state from previously (MAC seen where previously not, or
vice versa), then an update is performed against SmartThings using a http GET.

If there are any changes to the state then the current state is dumped to the
state file (I avoid writing every time due to some router boxes running on
limited write flash filesystems).

## Config

Normally the config is saved in the json state file, however originally its
loaded from a CSV which has 4 fields:

1. `name` - The name of the victim
2. `mac` - The MAC address of their identifying device
3. `appid` - The App ID of the SmartThings SmartApp (see write up from @fuzzysb )
4. `token` - The authentication token for the SmartApp

## Options

- `-load file.csv` - the config csv
- `-baseurl url` - the SmartThings base URL
- `-force` - always send updates to SmartThings even if no change
- `-state file.json` - the state file - default `~/.presence.json`

The Base URL can only be set on a config `-load` - by default it is the 
European endpoint - `https://graph-eu01-euwest1.api.smartthings.com`

## Using

Having done the initial setup, set it to run regularly from cron.

I run it every minute at present - the ARP timeout is 20 minutes, so although
people coming in (so changing from `away` to `home`) will be picked up in a
minute as their phone attaches to local wifi, when they leave it will take 20
minutes until they are marked `away`

It appears Android phones can occaisionally fall off the network even when the
person is within range all the time. iPhones do not appear to do this, although
I have seen a couple of unexplained additional arrival events.

I'm still playing with this.

## Go Baby Talk

Not written anything in go previously.  Its likely to have issues!
