# syntax=docker/dockerfile:1

FROM migrate/migrate:v4.17.1

# Bundle migrations into the image for Kubernetes Job usage.
COPY migrations /migrations

ENTRYPOINT ["migrate"]
