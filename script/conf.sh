Target="streamf"
Docker="king011/streamf"
Dir=$(cd "$(dirname $BASH_SOURCE)/.." && pwd)
Version="v0.0.9"
Platforms=(
    darwin/amd64
    windows/amd64
    linux/arm
    linux/amd64
)