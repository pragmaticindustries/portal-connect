docker build -t portal-connect .
docker run --env-file=.env portal-connect
