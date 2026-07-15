# GO BASE

A base set of packages for building go applications.

- [GO BASE](#go-base)
  - [👀 Usage](#-usage)
    - [Devcontainer](#devcontainer)
      - [Post-create Script Helpers](#post-create-script-helpers)
      - [Base Container Images](#base-container-images)
    - [Environment Variables](#environment-variables)
      - [General](#general)
      - [HTTP Client](#http-client)
      - [Logging](#logging)
  - [⚠️ License](#️-license)
  - [🤝 Contact](#-contact)

## 👀 Usage

### Devcontainer

#### Post-create Script Helpers

In `.devcontainer/postcreate.scripts.d`, there are various scripts that can be run from a single devcontainer
post-create script to install various tools and services.

#### Base Container Images

The `.devcontainer/*.Dockerfile` Dockerfiles build containers that can be used as a base for creating a specialised
devcontainer for a project. Packages for each image built from these will be available at
[packages](https://github.com/orgs/immanent-tech/packages?repo_name=go-base).

### Environment Variables

#### General

- `APP_NAME`: the application name. i.e., "My App".
- `APP_ID`: the application id in reverse-dotted notation. i.e., "com.my.app".
- `APP_DESCRIPTION`: the application description. i.e., what the app does.
- `APP_VERSION`: the application version. Defaults to either the current git tag or "_UNKNOWN_" if the git tag cannot be
  parsed.
- `APP_ENVIRONMENT`: the application envrionment, either "production" or "development". Defaults to "development".
- `APP_BASEURL`: the base URL on which the web component of the app will run.

#### HTTP Client

- `CLIENT_USERAGENT`: the user-agent string to send with client requests. Defaults to `${APP_NAME}/${APP_VERSION}`.
- `CLIENT_REQUEST_TIMEOUT`: the default timeout for requests. No default. Parsed as a duration string, e.g., "30s".
- `CLIENT_REQUEST_RETRIES`: the number of retries for a failed request. Defaults to 3.

#### Logging

- `LOG_LEVEL`: the default log level. Defaults to "info". One of "trace", "debug", "info", "warn" or "error".
- `LOG_FILE`: a file to write logs to. Will be overwritten if it exists. When running in a container, setting this value
  will have no effect (i.e., writing logs to files is disabled).

## ⚠️ License

Distributed under the AGPL-3.0-or-later License. See [LICENSE](https://github.com/immanent-tech/go-base/blob/main/LICENSE) for more information.

## 🤝 Contact

Immanent Tech — [hello@immanent.tech](mailto:hello@immanent.tech)

Project Link: [github.com/immanent-tech/go-base](https://github.com/immanent-tech/go-base)
