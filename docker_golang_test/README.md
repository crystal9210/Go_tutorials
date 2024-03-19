docker build . -t go-containerized-api:latest
docker run -e PORT=9000 -p 9000:9000 go-containerized-api:latest
【参照】
https://www.youtube.com/watch?v=C5y-14YFs_8
