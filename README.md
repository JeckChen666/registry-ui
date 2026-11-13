# Registry UI

[![Go Report Card](https://goreportcard.com/badge/github.com/quiq/registry-ui)](https://goreportcard.com/report/github.com/quiq/registry-ui)

A lightweight, fast web interface for browsing and managing Docker Registry and OCI-compatible registries.

Docker image: [quiq/registry-ui](https://hub.docker.com/r/quiq/registry-ui/tags/)

### Features

- Browse repositories, tags, and nested repository trees at any depth
- View full image details: layers, config, and command history
- Support for Docker and OCI image formats, including multi-platform image indexes
- Event listener to capture registry notifications, stored in SQLite or MySQL
- Built-in tag retention with CLI and in-process cron scheduling
- Auto-discovery of authentication methods (basic auth, token service, keychain, etc.)
- Repository list and tags are cached and refreshed in the background
- React interface built with shadcn/ui and Tailwind CSS

The UI supports optional password authentication. It does not terminate TLS; use a reverse proxy when HTTPS is required.

### Quick start

Start a Docker registry on your host (if you don't already have one):

    docker run -d --network host --name registry registry

Push any image to `127.0.0.1:5000/owner/name`:

    docker tag alpine:edge 127.0.0.1:5000/owner/name
    docker push 127.0.0.1:5000/owner/name

Run Registry UI and open http://127.0.0.1:8000 in your browser:

    docker run --rm --network host \
        -e REGISTRY_HOSTNAME=127.0.0.1:5000 \
        -e REGISTRY_INSECURE=true \
        --name registry-ui quiq/registry-ui

### Local development

The Go server serves the built frontend from `static/ui`. Build it before starting the server:

    cd frontend
    npm ci
    npm run build
    cd ..
    go run . -config-file config.yml

Rebuild the frontend after changing its source. The Docker image builds the frontend automatically; a local Node.js installation is not needed for Docker builds.

### Configuration

Configuration is stored in `config.yml` with self-descriptive options. Any option can be
overridden via environment variables using `SECTION_KEY_NAME` syntax,
e.g. `LISTEN_ADDR`, `PERFORMANCE_TAGS_ASYNC_REFRESH_INTERVAL`, `REGISTRY_HOSTNAME`.

To pass a full config file:

    docker run -d -p 8000:8000 -v /local/config.yml:/opt/config.yml:ro quiq/registry-ui

To use a custom root CA certificate:

    -v /local/rootcacerts.crt:/etc/ssl/certs/ca-certificates.crt:ro

To persist the SQLite database for event data:

    -v /local/data:/opt/data

Ensure `/local/data` is owned by `nobody` (uid 65534 on Alpine).

The container supports `--read-only` mode. If you use the event listener, mount the data folder
in read-write mode as shown above so the SQLite database remains writable.

To set a custom timezone:

    -e TZ=America/Los_Angeles

### Event listener

To receive notification events, configure your Docker Registry:

    notifications:
      endpoints:
        - name: registry-ui
          url: http://registry-ui.local:8000/event-receiver
          headers:
            Authorization: [Bearer abcdefghijklmnopqrstuvwxyz1234567890]
          timeout: 1s
          threshold: 5
          backoff: 10s
          ignoredmediatypes:
            - application/octet-stream

Adjust the URL and token as appropriate.
If you run the UI under a non-default base path (e.g. `/ui`), use `/ui/event-receiver` as the endpoint path.

### Using MySQL instead of SQLite

To use MySQL, update `event_database_driver` and `event_database_location` in the config file.
Create the database referenced in the DSN beforehand. Required privileges: `SELECT`, `INSERT`, `DELETE`.

To create the tables manually (avoids granting `CREATE`):

	CREATE TABLE events (
		id INTEGER PRIMARY KEY AUTO_INCREMENT,
		action CHAR(4) NULL,
		repository VARCHAR(100) NULL,
		tag VARCHAR(100) NULL,
		ip VARCHAR(45) NULL,
		user VARCHAR(50) NULL,
		created DATETIME NULL
	);

	CREATE TABLE purge_runs (
		id INTEGER PRIMARY KEY AUTO_INCREMENT,
		started_at DATETIME NULL,
		finished_at DATETIME NULL,
		success BOOLEAN NULL,
		dry_run BOOLEAN NULL,
		cron_expr VARCHAR(100) NULL,
		repo_count INTEGER NULL,
		candidate_tag_count INTEGER NULL,
		deleted_tag_count INTEGER NULL,
		estimated_freed_bytes BIGINT NULL,
		summary TEXT NULL,
		error_message TEXT NULL,
		created DATETIME NULL
	);

### Tag purging

First, enable tag deletion in your Docker Registry config:

    storage:
      delete:
        enabled: true

Then configure the built-in scheduler in `config.yml`:

    purge_tags:
      cron: "10 3 * * *"
      dry_run: false
      log_retention_days: 30
      keep_days: 90
      keep_count: 10

Preview what would be purged with dry-run mode:

    docker exec -t registry-ui /opt/registry-ui -purge-tags -dry-run

You can still run the purge manually:

    docker exec -t registry-ui /opt/registry-ui -purge-tags

The scheduler uses the container's local timezone. Tag age is measured from the image creation time, not the time a tag was pushed. For multi-platform images, the newest platform creation time is used. Tags without a reliable creation time cause their repository to be skipped and the run to be marked as failed.

Purging deletes manifests by digest. If a digest is shared with a tag selected for retention, all of its tags are kept. The task checks the repository again before deleting and verifies that tags disappear afterwards. Registry APIs do not provide an atomic check-and-delete operation, so avoid pushing or retagging images in a repository while it is being purged. Multiple UI instances running the scheduler also require external coordination.

The size shown in the purge log is the sum of candidate image sizes, not the actual disk space recovered. Layers may be shared and registry garbage collection is typically required to reclaim space.

### Screenshots

Repository list:

![image](screenshots/1.png)

Tag list:

![image](screenshots/2.png)

Image Index info:

![image](screenshots/3.png)

Image info:

![image](screenshots/4.png)
