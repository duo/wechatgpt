# wechatgpt
A WeChat ChatGPT bot based on [openwechat](https://github.com/eatmoreapple/openwechat) and [chatgptauth](https://github.com/rodjunger/chatgptauth).

### Usage

*nix
```bash
TASK_TIMEOUT=120s CHATGPT_EMAIL=xxx CHATGPT_PASSWORD=yyy ./wechatgpt
```

windows
```bash
set TASK_TIMEOUT=120s
set CHATGPT_EMAIL=xxx
set CHATGPT_PASSWORD=yyy
wechatgpt.exe
```

docker

[lxduo/wechatgpt](https://hub.docker.com/r/lxduo/wechatgpt)

### Command
|   CMD    | Function                   |
| :------: | -------------------------- |
| `!reset` | Reset ChatGPT conversation |

### Environment
|      Variable      | Function                                                |
| :----------------: | ------------------------------------------------------- |
|  `CHATGPT_EMAIL`   | ChatGPT email                                           |
| `CHATGPT_PASSWORD` | ChatGPT password                                        |
| ~`SESSION_TOKEN`~  | ~ChatGPT __Secure-next-auth.session-token~ `deprecated` |
|   `TASK_TIMEOUT`   | ChatGPT API query timeout duration                      |
|   `AUTO_ACCEPT`    | Auto accept WeChat friend request                       |
