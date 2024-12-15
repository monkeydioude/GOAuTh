#!/bin/sh

containerName="goauth-tests-db-auto"

waitForContainerReadiness()
{
    i=1
    maxI=5

    if [ -z "$(docker ps | grep $containerName)" ]; then
        echo "[ERR ] Could not start PGSQL container. Exiting"
        exit 1
    fi
    while [ $i -le $maxI ]; do
        if [ ! -z "$(docker exec $containerName sh -c 'pg_isready -U test -d test_db' | grep 'accepting')" ]; then
            break
        fi
        echo "[WARN] PGSQL container not ready ($i/$maxI)..."
        i=$((i+1))
        sleep 1
    done
    if [ $i = $maxI ]; then
        echo "[ERR ] Could not get PGSQL container ready. Exiting"
        exit 1
    fi 
}

echo "[INFO] Starting '$containerName' PGSQL test container!"
docker run -d --rm -p 5444:5432 --name $containerName --env POSTGRES_USER=test --env POSTGRES_PASSWORD=test --env POSTGRES_DB=test_db postgres

echo "[INFO] Waiting for '$containerName' PGSQL test container readiness!"
waitForContainerReadiness

echo "[INFO] '$containerName' is ready! Lezgong tests!"
DB_PATH=postgres://test:test@0.0.0.0:5444/test_db go test -count=1 -v ./...
exit_code=$?

echo "[INFO] Stopping '$containerName' PGSQL test container!"
docker stop $containerName

exit $exit_code