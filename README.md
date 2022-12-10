# wechatgpt
A WeChat ChatGPT bot based on [openwechat](https://github.com/eatmoreapple/openwechat).

### Usage

*nix
```bash
TASK_TIMEOUT=120s SESSION_TOKEN=xxxxxxxxx ./wechatgpt
```

windows
```bash
set TASK_TIMEOUT=120s
set SESSION_TOKEN=xxxxxxxxx
wechatgpt.exe
```

docker

[lxduo/wechatgpt](https://hub.docker.com/r/lxduo/wechatgpt)

### Command
|   CMD    | Function                   |
| :------: | -------------------------- |
| `!reset` | Reset ChatGPT conversation |

### Environment
|    Variable     | Function                                 |
| :-------------: | ---------------------------------------- |
| `SESSION_TOKEN` | ChatGPT __Secure-next-auth.session-token |
| `TASK_TIMEOUT`  | ChatGPT API query timeout duration       |
|  `AUTO_ACCEPT`  | Auto accept WeChat friend request        |
