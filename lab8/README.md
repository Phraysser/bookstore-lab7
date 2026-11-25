# Лабораторная работа № 8 — Основы Docker

Вариант 9. Книжный магазин и Nginx как обратный прокси.

Workflow `.github/workflows/lab8.yml` запускается при push в ветку `lab8/docker`. Он выполняет практику в чистом окружении Ubuntu с Docker Engine. После сборки образ публикуется как `phraysser/bookstore:v1`, а следующий job скачивает и проверяет его на другом runner.

Команды этапов находятся в `lab8/docker-lab.sh`. Этап `cleanup` рассчитан на одноразовый runner, поскольку он очищает и кэш Docker builder. На личной машине удаляйте только ресурсы своей лабораторной.

## Запуск только варианта 9

Из корня проекта в Bash с работающим Docker Engine:

```bash
docker build -t bookstore:v1 .
docker network create appnet
docker run -d --name api --network appnet bookstore:v1
docker run -d --name proxy --network appnet -p 127.0.0.1:8080:80 \
  --mount "type=bind,src=$PWD/lab8/nginx.conf,dst=/etc/nginx/conf.d/default.conf,readonly" \
  nginx:1.28-alpine

curl -i -H 'Content-Type: application/json' \
  -d '{"title":"Go in Action","author":"William Kennedy","price":39.9,"stock":3}' \
  http://localhost:8080/api/books
```

Подставьте UUID из ответа в URL `http://localhost:8080/api/books/UUID`. Запрос проходит через Nginx, что видно по ответному заголовку `X-Bookstore-Proxy: nginx`. Конечный слеш в `proxy_pass http://api:8080/` заменяет совпавший префикс `/api/` на `/`.

Очистка ресурсов этого сценария:

```bash
docker rm -f proxy api
docker network rm appnet
```
