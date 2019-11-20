#!/bin/bash

if [[ -z "$PGPASSWORD" ]]
then
  echo "⛔ empty PGPASSWORD"
  exit 1
fi

if [[ -z "$PGHOST" ]]
then
  echo "⛔ empty PGHOST"
  exit 1
fi

if [[ -z "$PGDATABASE" ]]
then
  echo "⛔ empty PGDATABASE"
  exit 1
fi

if [[ -z "$PGUSER" ]]
then
  echo "⛔ empty PGUSER"
  exit 1
fi

if [[ -z "$PGPORT" ]]
then
  echo "⛔ empty PGPORT"
  exit 1
fi

# check
echo "➡️  check connect"
psql -q -h $PGHOST -d $PGDATABASE -U $PGUSER -p $PGPORT -c "SELECT version();"
[ $? -eq 0 ]  || { echo "⛔ check connect - FAIL"; exit 1 ;}

for f in $*
do
    echo "➡️  $f"
    psql -q -h $PGHOST -d $PGDATABASE -U $PGUSER -p $PGPORT -v "ON_ERROR_STOP=1" -f $f
    [ $? -eq 0 ]  || { echo "⛔ $f - FAIL"; exit 1 ;}
done
