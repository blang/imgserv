imgserv
======

imgserv is a simple service which accepts an image via http post and makes it accessible via get.

It's a simple solution e.g. to provide access to webcam shots. The upload and view routes are secured via simple basic auth and token auth.

Usage
-----
```bash
$ go get github.com/blang/imgserv
```

```bash
$ ./imgserv --help

Usage of ./imgserv:
  -listen string
        Listen (default ":12345")
  -password string
        Password for basic auth
  -path string
        Filepath to image (default "/tmp/img.jpg")
  -uploadtoken string
        Token for upload
  -username string
        Username for basic auth

$ ./imgserv -listen ":12345" -username "webcam" -password "access" -path /tmp/webcam.jpg -uploadtoken "myaccesstoken"
```

Update image:
```bash
curl -F file=@/home/user/webcam.jpg '127.0.0.1:12345/upload?token=myaccesstoken'
```

View image in browser: http://127.0.0.1:12345 using given username and password.

License
-----

See [LICENSE](LICENSE) file.
