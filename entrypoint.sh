#!/bin/sh

printf "\n ========================================="
printf "\n ========================================="
printf "\n === CLOUDFLARE DNS OVER TLS CONTAINER ==="
printf "\n ========================================="
printf "\n ========================================="
printf "\n == by github.com/qdm12 - Quentin McGaw ==\n\n"

printf "\nUnbound version: $(unbound -h | grep "Version" | cut -d" " -f2)"
printf "\nVerbosity level set to $VERBOSITY"
printf "\nVerbosity details level set to $VERBOSITY_DETAILS"
printf "\nMalicious hostnames blocking is $BLOCK_MALICIOUS\n"
[[ "$VERBOSITY" == "" ]] || sed -i "s/verbosity: 0/verbosity: $VERBOSITY/g" /etc/unbound/unbound.conf
$(grep -Fq 127.0.0.1 /etc/resolv.conf) || echo "WARNING: The domain name is not set to 127.0.0.1 so the healthcheck will likely be irrelevant!"
[[ "$VERBOSITY_DETAILS" == "" ]] || [[ "$VERBOSITY_DETAILS" == "0" ]] || ARGS=-$(for i in `seq $VERBOSITY_DETAILS`; do printf "v"; done)
touch /etc/unbound/blocks-malicious.conf
[[ "$BLOCK_MALICIOUS" != "on" ]] || (printf "Extracting blocks-malicious.conf.bz2..."; tar -xjf /etc/unbound/blocks-malicious.conf.bz2 -C /etc/unbound/; printf "DONE\n")
unbound -d $ARGS
status=$?
printf "\n ========================================="
printf "\n Unbound exited with status $status"
printf "\n =========================================\n"