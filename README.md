# Gitto

Gitto is a service for creating anonymous git repos.

Gitto provide an API to create git repos with a secret URL. Anyone
who has access to the URL can read and write to the repo.

Gitto also supports webhook per repository. The webhook is triggered on push events. They are implemented as post-receive hook in git. As of now only one hook is supported per repo.

## How to run

**Setup**

Set the root directory for repos.

```
export GITTO_ROOT=/var/git
export GITTO_API_TOKEN=4CKzyyU46Bvh0QDonhaKFWULtrGBKh3F
```

The `GITTO_ROOT` defaults to `git` when not specified.

**Build**

Build the project using:

```
$ go build
```

**Run**

Run the server using:


```
$ ./gitto
```

## The API

### Create a new repo

```
POST /api/repos

{
    "name": "rajdhani"
}
---
200 OK
{
    "id": "abcd12345678",
    "name": "rajdhani",
    "git_url": "https://example.com/abcd12345678/rajdhani.git",
}
```

### Get repo info

```
GET /api/repos/abcd12345678

{
    "id": "abcd12345678",
    "name": "rajdhani",
    "git_url": "https://example.com/abcd12345678/rajdhani.git",
}
```

### Delete a repo

```
DELETE /api/repos/abcd12345678
---
200 OK
{}
```

### Get webhook

```
GET /api/repos/abcd12345678/hook

{"url": "https://example.com/foo"}
```

### Set webhook

```
POST /api/repos/abcd12345678/hook

{"url": "https://example.com/foo"}
```
