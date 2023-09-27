#!/bin/sh

# $1 - IP
# $2 - Port
# $3 - Protocol

set -e

echo $1 | xargs -n3 sh -c 'iptables -t filter -I FORWARD -p $3 --dport $2 -j ACCEPT' sh
echo $1 | xargs -n3 sh -c 'iptables -t nat -I PREROUTING -p $3 --dport $2 -j DNAT --to $1:$2' sh
echo $1 | xargs -n3 sh -c 'iptables -t nat -I POSTROUTING -d $1/32 -p $3 -j MASQUERADE' sh

sleep infinity