# Лабораторная работа № 7 — CI/CD для книжного магазина

[![CI/CD](https://github.com/Phraysser/bookstore-lab7/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/Phraysser/bookstore-lab7/actions/workflows/ci.yml)

Вариант 9. GitHub Actions проверяет Go-сервис из ЛР № 6, собирает Docker-образ, разворачивает его в kind-кластере на runner и проверяет HTTP. При push в main или push тега v* успешно проверенный образ публикуется в Docker Hub. На pull request развёртывание тоже проверяется, а публикация пропускается.

## 1. Подготовить GitHub и Docker Hub

- Создай пустой репозиторий `bookstore-lab7` на GitHub. Публичный репозиторий позволяет выполнить защиту ветки на GitHub Free. Группу, преподавателя и правила доступа сверяй с требованиями своей лабораторной.
- В Docker Hub создай публичный репозиторий `bookstore` в своём пространстве имён и персональный токен с правом записи.
- В GitHub-репозитории открой **Settings → Secrets and variables → Actions → New repository secret**. Добавь `DOCKERHUB_USERNAME` (имя пользователя Docker Hub) и `DOCKERHUB_TOKEN` (токен Docker Hub).

## 2. Проверить проект и отправить первый коммит

В Git Bash открой корень распакованного проекта, где находятся `go.mod` и `Dockerfile`. Для первичной проверки нужен Go 1.23 или новее.

```bash
go version
go build ./...
go test -v -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

Подставь свой логин GitHub. Если используется уже настроенный SSH-доступ, в качестве origin можно указать SSH-адрес из меню **Code** репозитория.

```bash
github_owner='ВСТАВЬ_ЛОГИН_GITHUB'
git init -b main
git add .
git status --short
git commit -m "Add bookstore CI/CD and kind deployment"
git remote add origin "https://github.com/$github_owner/bookstore-lab7.git"
git push -u origin main
```

Если это уже Git-репозиторий, проверь `git remote -v` и пропусти `git init` и `git remote add`, когда они уже выполнены для нужного репозитория.

Для бейджей добавь строки ниже в README своего репозитория, подставив реальный логин. Docker-развёртывание отображается как job внутри CI, поэтому бейдж общего workflow охватывает оба этапа.

```markdown
![CI/CD](https://github.com/OWNER/bookstore-lab7/actions/workflows/ci.yml/badge.svg?branch=main)
```

Если добавляешь бейдж после первого push, сделай коммит и push до настройки защиты. Этот повторный запуск также пригодится для сравнения кэша.

```bash
git add README.md
git commit -m "Add CI status badge"
git push
```

## 3. Проверить Actions и Docker Hub

В **Actions → CI** открой запуск от push. Ожидаются две проверки **Build and Test (Go 1.23)** и **Build and Test (Go 1.24)**. В каждой открой шаги **Lint**, **Build**, **Test** и **Upload coverage**. Затем открой **Docker / Build, deploy to kind and publish** и проверь строки `deployment.apps/bookstore created`, `service/bookstore created`, успешный rollout, готовый Pod и вывод **Check HTTP in kind**.

В Docker Hub на вкладке **Tags** проверь `main` и `sha-…`. При необходимости можно проверить выпуск версии командой `git tag v1.0.0 && git push origin v1.0.0` после успешной проверки main. Этот запуск опубликует `v1.0.0` и SHA-тег.

## 4. Проверить защиту main и реакцию на ошибку

После первого зелёного запуска в **Settings → Branches** добавь правило для `main`. Включи **Require a pull request before merging** и **Require status checks to pass before merging**. Для индивидуальной работы оставь число обязательных одобрений равным нулю. Выбери обе проверки **Build and Test**. Если GitHub показывает отключение обхода правила администраторами, включи его для демонстрации блокировки.

```bash
git switch -c lab7/ci-failure
cat > ci_failure.go <<'EOF'
package main

func demoFailure() {
    unused := 42
}
EOF
git add ci_failure.go
git commit -m "Demonstrate CI failure"
git push -u origin lab7/ci-failure
```

На GitHub открой PR из `lab7/ci-failure` в `main`. Дождись красных проверок. Открой шаг **Lint** и найди диагностику о неиспользуемой переменной. Сделай кадр PR с заблокированным слиянием. Затем удали только демонстрационный файл:

```bash
git rm ci_failure.go
git commit -m "Fix demonstration build error"
git push
```

В том же PR дождись успешных проверок и выполни слияние. Покажи зелёные проверки и статус **Merged**. После слияния push в `main` запускает публикацию заново.

## 5. Скачать coverage и сравнить кэш

Сравни два запуска одной версии Go. В шаге **Cache Go modules and build** найди попадание в кэш, а в **Download dependencies** сравни фактическую длительность. Ускорение зависит от runner, поэтому в отчёт перенеси измеренное время.

На странице успешного запуска открой **Artifacts** и скачай `coverage-go-1.24`. Распакуй ZIP и из каталога с исходниками выполни, указав фактический путь к скачанному файлу:

```bash
go tool cover -html="/путь/к/coverage.out" -o coverage.html
```

Открой `coverage.html` в браузере. Для выявления возможных гонок CI запускает тесты с флагом `-race`.

## 6. Скачать и запустить опубликованный образ

Нужен работающий Docker Engine или Docker Desktop. Укажи **логин Docker Hub**, который может отличаться от GitHub. Порты хоста 18080 и 19090 перенаправлены на стандартные порты сервиса 8080 и 9090 внутри контейнера.

```bash
dockerhub_user='ВСТАВЬ_ЛОГИН_DOCKER_HUB'
image="$dockerhub_user/bookstore:main"
docker pull "$image"
docker run -d --rm --name bookstore-lab7 \
  -p 127.0.0.1:18080:8080 -p 127.0.0.1:19090:9090 "$image"
docker ps --filter name=bookstore-lab7

curl -i http://127.0.0.1:18080/books \
  -H 'Content-Type: application/json' \
  -d '{"title":"Go in Action","author":"William Kennedy","price":39.9,"stock":3}'
```

Скопируй UUID из `201 Created` и прочитай книгу. Ожидается `200 OK`, `"stock":3` и `"in_stock":true`.

```bash
id='ВСТАВЬ_UUID'
curl -i "http://127.0.0.1:18080/books/$id"
```

Когда сделал скриншоты, останови контейнер командой `docker stop bookstore-lab7`. Если старый процесс ЛР № 6 занял 18080 или 19090, останови его или выбери другие порты хоста и обнови URL curl.

## 7. Скриншоты для отчёта

1. Вкладка Actions, обе версии Go и успешные шаги Lint, Build и Test.
2. Отрицательная проверка PR, диагностика Lint, блокировка Merge и успешный результат после исправления.
3. Восстановление кэша и артефакт coverage либо открытый HTML-отчёт.
4. Шаг Deploy to kind, успешный rollout, готовый Pod, Service и вывод Check HTTP in kind.
5. Теги образа в Docker Hub.
6. Локальный `docker ps` и ответы curl при создании и чтении книги.
7. Правило защиты main и зелёный бейдж README, если они не видны в предыдущих кадрах.

Сохрани ссылки на GitHub-репозиторий, успешный запуск Actions, тестовый PR и страницу тегов Docker Hub.
