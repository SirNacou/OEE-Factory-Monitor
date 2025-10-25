set windows-shell := ["powershell", "-Command"]
set shell := ["bash", "-c"]

default:

docker up:
  docker compose up -d --build