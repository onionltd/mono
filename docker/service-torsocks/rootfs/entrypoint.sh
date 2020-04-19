#!/bin/sh

if [ -z "$TORSOCKS_TOR_ADDRESS" ]; then
  echo "missing TORSOCKS_TOR_ADDRESS"
  exit 1
fi

if [ -z "$TORSOCKS_TOR_PORT" ]; then
  echo "missing TORSOCKS_TOR_PORT"
  exit 1
fi

cat <<EOF > "$TORSOCKS_CONF_FILE"
TorAddress $TORSOCKS_TOR_ADDRESS
TorPort $TORSOCKS_TOR_PORT
EOF

SOCKS_PROXY="socks5://$TORSOCKS_TOR_ADDRESS:$TORSOCKS_TOR_PORT"
export HTTP_PROXY="$SOCKS_PROXY"
export HTTPS_PROXY="$SOCKS_PROXY"
export ALL_PROXY="$SOCKS_PROXY"

torsocks "$@"
