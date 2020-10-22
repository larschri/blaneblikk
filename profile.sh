#!/bin/bash -eu

PORT=1392
OUTPUT=profile.svg
SECONDS=15

(go run . --address=localhost:${PORT} 2>&1 &) | grep -q "Listening to"

(
	while true; do
		curl "http://localhost:${PORT}/bb?lat0=60.16542574699484&lng0=10.393753051757814&lat1=59.97039127513498&lng1=10.156130790710451" >& /dev/null
	done

) &

docker run --rm --network="host" uber/go-torch -u http://localhost:${PORT}/debug/pprof -p -t=$SECONDS > $OUTPUT

kill -- -$$
