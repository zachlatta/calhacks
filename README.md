Required configuration:

Put in `config/config.yml`:

```
GITHUB_CLIENT_ID: # TODO
GITHUB_CLIRENT_SECRET: # TODO
REDIRECT_URL: # TODO

JWT_SECRET: # TODO
```

You must also set `DATABASE_URL` in the environment when running in production.

## Usage

Start application:

    $ fig up

Run DB migrations:

    $ fig run web goose --path="../../db" up
