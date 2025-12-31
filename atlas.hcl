variable "envfile" {
  type    = string
  default = "./.env"
}

locals {
  envfile = {
    for line in split("\n", file(var.envfile)) : split("=", line)[0] => regex("=(.*)", line)[0]
    if !startswith(line, "#") && length(split("=", line)) > 1
  }
}

data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "./internal/entity",
    "--dialect", "postgres"
  ]
}
env "gorm" {
  src = data.external_schema.gorm.url
  dev = "postgres://${local.envfile["DB_USER"]}:${local.envfile["DB_PASSWORD"]}@${local.envfile["DB_HOST"]}:${local.envfile["DB_PORT"]}/${local.envfile["DB_NAME"]}?sslmode=${local.envfile["DB_SSL_MODE"]}"
  migration {
    dir = "file://migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}