# go-jira-tui

It's a TUI for JIRA. Written in Go.

## config

```yaml
---
email: ""
token: ""
url: "https://guppy0130.atlassian.net"
```

## usage

```bash
cat << EOF > config.yml  # or $XDG_CONFIG_DIR/go-jira-tui/config.yml
---
email: ""
token: ""
url: ""
EOF

go build
./go-jira-tui
```
