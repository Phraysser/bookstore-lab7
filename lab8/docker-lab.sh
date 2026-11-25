#!/usr/bin/env bash
set -euo pipefail
set -x

nginx=nginx:1.28-alpine
pg=postgres:16
hub=phraysser/bookstore:v1
book='{"title":"Go in Action","author":"William Kennedy","price":39.9,"stock":3}'

wait_http() {
    local url=$1
    for attempt in {1..40}; do
        if curl -sS --max-time 2 -o /dev/null "$url" 2>/dev/null; then
            return
        fi
        sleep 1
    done
    return 1
}

wait_pg() {
    local name=$1
    for attempt in {1..60}; do
        if docker exec "$name" pg_isready -h 127.0.0.1 -U lab -d lab >/dev/null 2>&1; then
            return
        fi
        sleep 1
    done
    docker logs "$name" >&2
    return 1
}

check_book() {
    local url=$1 marker=${2:-} id work
    work=$(mktemp -d)
    wait_http "$url/missing"
    curl --fail-with-body -sS -D "$work/headers" -o "$work/book.json" \
        -H 'Content-Type: application/json' -d "$book" "$url"
    cat "$work/headers" "$work/book.json"
    grep -q '^HTTP/1\.[01] 201' "$work/headers"
    if [[ -n "$marker" ]]; then
        grep -qi '^X-Bookstore-Proxy: nginx' "$work/headers"
    fi
    id=$(jq -er '.id | select(length > 0)' "$work/book.json")
    curl --fail-with-body -sS -D "$work/headers" -o "$work/book.json" "$url/$id"
    cat "$work/headers" "$work/book.json"
    grep -q '^HTTP/1\.[01] 200' "$work/headers"
    jq -e --arg id "$id" '.id == $id and .stock == 3 and .in_stock == true' "$work/book.json"
    local status
    status=$(curl -sS -o /dev/null -w '%{http_code}' -X PATCH \
        -H 'Content-Type: application/json' -d '{"delta":-3}' "$url/$id/stock")
    echo "PATCH stock: HTTP $status"
    test "$status" = 204
    curl -fsS "$url/$id" | tee "$work/book.json"
    jq -e '.stock == 0 and .in_stock == false' "$work/book.json"
    rm -r "$work"
}

case "${1:?specify a lab stage}" in
    engine)
        docker version
        docker info --format 'Server={{.ServerVersion}} Driver={{.Driver}} OS={{.OperatingSystem}}'
        docker run --name lab8-hello hello-world
        ;;
    images)
        docker pull nginx
        docker pull "$nginx"
        docker image ls nginx
        docker image inspect --format 'ID={{.Id}} Size={{.Size}} bytes' nginx:latest
        docker image rm nginx:latest
        docker run -d --name my-nginx "$nginx"
        docker ps --filter name=my-nginx
        docker logs my-nginx
        docker stop my-nginx
        docker ps -a --filter name=my-nginx
        docker start my-nginx
        docker ps --filter name=my-nginx
        docker rm -f my-nginx
        ;;
    ports)
        docker run -d --name web --hostname web -p 127.0.0.1:8080:80 "$nginx"
        docker port web
        wait_http http://127.0.0.1:8080
        curl -fsS http://localhost:8080 | grep '<title>'
        docker exec web hostname
        docker exec web ls /etc/nginx
        docker rm -f web
        ;;
    volumes)
        dir=$(mktemp -d)
        printf '<h1>Version 1 from host</h1>\n' > "$dir/index.html"
        chmod 755 "$dir"
        docker run -d --name web-bind -p 127.0.0.1:8081:80 \
            --mount "type=bind,src=$dir,dst=/usr/share/nginx/html,readonly" "$nginx"
        wait_http http://127.0.0.1:8081
        curl -fsS http://127.0.0.1:8081 | grep 'Version 1 from host'
        printf '<h1>Version 2 from host</h1>\n' > "$dir/index.html"
        curl -fsS http://127.0.0.1:8081 | grep 'Version 2 from host'
        docker rm -f web-bind
        rm -r "$dir"

        docker volume create pgdata
        docker volume inspect pgdata
        docker run -d --name pg-original -e POSTGRES_USER=lab -e POSTGRES_DB=lab \
            -e POSTGRES_HOST_AUTH_METHOD=trust -v pgdata:/var/lib/postgresql/data "$pg"
        wait_pg pg-original
        docker exec pg-original psql -v ON_ERROR_STOP=1 -U lab -d lab \
            -c 'CREATE TABLE books_demo (id integer PRIMARY KEY, title text NOT NULL);' \
            -c "INSERT INTO books_demo VALUES (1, 'Go in Action');" \
            -c 'SELECT * FROM books_demo;'
        docker rm -f pg-original
        docker run -d --name pg-recreated -e POSTGRES_USER=lab -e POSTGRES_DB=lab \
            -e POSTGRES_HOST_AUTH_METHOD=trust -v pgdata:/var/lib/postgresql/data "$pg"
        wait_pg pg-recreated
        docker exec pg-recreated psql -v ON_ERROR_STOP=1 -U lab -d lab -c 'SELECT * FROM books_demo;'
        row=$(docker exec pg-recreated psql -U lab -d lab -At -c 'SELECT title FROM books_demo WHERE id = 1;')
        test "$row" = 'Go in Action'
        ;;
    networks)
        docker network create mynet
        docker run -d --name nginx1 --network mynet "$nginx"
        docker run -d --name client --network mynet alpine:3.22 sleep 120
        docker exec client wget -qO- http://nginx1 | grep '<title>'
        docker network inspect mynet
        ;;
    build)
        dir=$(mktemp -d)
        tar --exclude=.git -cf - . | tar -xf - -C "$dir"
        rm "$dir/.dockerignore"
        docker build --progress=plain -t bookstore:no-ignore "$dir" 2>&1 | tee "$dir.before.log"
        docker build --progress=plain -t bookstore:v1 . 2>&1 | tee "$dir.after.log"
        docker image ls bookstore
        docker image inspect --format '{{index .RepoTags 0}}: {{.Size}} bytes' bookstore:no-ignore bookstore:v1
        echo 'Context size before and after .dockerignore:'
        grep 'transferring context:' "$dir.before.log"
        grep 'transferring context:' "$dir.after.log"
        rm -r "$dir" "$dir.before.log" "$dir.after.log"
        ;;
    bookstore)
        docker run -d --name bookstore-direct -p 127.0.0.1:8082:8080 bookstore:v1
        docker ps --filter name=bookstore-direct
        check_book http://127.0.0.1:8082/books
        docker rm -f bookstore-direct
        docker network create appnet
        docker run -d --name db --network appnet -e POSTGRES_USER=lab -e POSTGRES_DB=lab \
            -e POSTGRES_HOST_AUTH_METHOD=trust "$pg"
        wait_pg db
        docker run -d --name api --network appnet --dns-opt ndots:0 bookstore:v1
        docker exec api nslookup -type=A db. 127.0.0.11
        docker network inspect --format '{{range .Containers}}{{.Name}} {{.IPv4Address}}{{println}}{{end}}' appnet
        ;;
    proxy)
        docker run -d --name proxy --network appnet --dns-opt ndots:0 -p 127.0.0.1:8080:80 \
            --mount "type=bind,src=$PWD/lab8/nginx.conf,dst=/etc/nginx/conf.d/default.conf,readonly" "$nginx"
        docker exec proxy nginx -t
        docker inspect --format '{{range .Mounts}}{{.Type}} {{.Destination}} writable={{.RW}}{{println}}{{end}}' proxy
        docker ps --filter network=appnet --format 'table {{.Names}}\t{{.Ports}}'
        check_book http://localhost:8080/api/books nginx
        test "$(curl -sS -o /dev/null -w '%{http_code}' http://localhost:8080/books)" = 404
        docker logs --tail 6 proxy
        ;;
    publish)
        test "${DOCKERHUB_USERNAME,,}/bookstore:v1" = "$hub"
        docker tag bookstore:v1 "$hub"
        docker push "$hub"
        ;;
    cleanup)
        echo 'Docker disk usage before cleanup'
        docker system df
        docker rm -fv lab8-hello pg-recreated nginx1 client api db proxy
        docker volume rm pgdata
        docker network rm mynet appnet
        docker image rm bookstore:no-ignore bookstore:v1 "$hub"
        docker builder prune --all --force
        echo 'Docker disk usage after cleanup'
        docker system df
        ;;
    pull)
        docker pull "$hub"
        docker run -d --name bookstore-pulled -p 127.0.0.1:8082:8080 "$hub"
        trap 'docker rm -f bookstore-pulled' EXIT
        docker ps --filter name=bookstore-pulled
        check_book http://127.0.0.1:8082/books
        ;;
    *)
        echo "Unknown stage: $1" >&2
        exit 2
        ;;
esac
