### Linux

```shell
curl -L https://github.com/jenkins-x/jx-project/releases/download/v{{.Version}}/jwizard-linux-amd64.tar.gz | tar xzv 
sudo mv jwizard /usr/local/bin
```

### macOS

```shell
curl -L  https://github.com/jenkins-x/jx-project/releases/download/v{{.Version}}/jwizard-darwin-amd64.tar.gz | tar xzv
sudo mv jwizard /usr/local/bin
```

