export SEUS_DBHOST="localhost"
export SEUS_DBPORT="8081"
export SEUS_DBUSER="test"
export SEUS_DBPASS="password"
export SEUS_DBNAME="seus"
export SEUS_SSLCRT="$PWD/debug/localhost.crt"
export SEUS_SSLKEY="$PWD/debug/localhost.key"

go run ./main.go
