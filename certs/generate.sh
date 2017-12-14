#!/bin/bash
# Regenerate the self-signed certificate for local host.

openssl req -x509 -sha256 -nodes -newkey ec:<(openssl ecparam -name secp256r1) -days 1024 -keyout demo.key -out demo.crt

