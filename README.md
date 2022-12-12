# wechatgpt
A WeChat ChatGPT bot based on [openwechat](https://github.com/eatmoreapple/openwechat) and [chatgptauth](https://github.com/rodjunger/chatgptauth).

### Usage

*nix
```bash
TASK_TIMEOUT=120s SESSION_TOKEN=xxx CF_CLEARANCE=xxx USER_AGENT=xxx ./wechatgpt
```

windows
```bash
set TASK_TIMEOUT=120s
set SESSION_TOKEN=xxx
set CF_CLEARANCE=xxx
set USER_AGENT=xxx
wechatgpt.exe
```

docker

[lxduo/wechatgpt](https://hub.docker.com/r/lxduo/wechatgpt)

### Command
|   CMD    | Function                   |
| :------: | -------------------------- |
| `!reset` | Reset ChatGPT conversation |

### Environment
|      Variable      | Function                                          |
| :----------------: | ------------------------------------------------- |
|  `SESSION_TOKEN`   | ChatGPT cookie `__Secure-next-auth.session-token` |
|   `CF_CLEARANCE`   | ChatGPT cookie `cf_clearance`                     |
|    `USER_AGENT`    | Browser user agent                                |
|   `TASK_TIMEOUT`   | ChatGPT API query timeout duration                |
|   `AUTO_ACCEPT`    | Auto accept WeChat friend request                 |
|  `CHATGPT_EMAIL`   | ChatGPT email                                     |
| `CHATGPT_PASSWORD` | ChatGPT password                                  |
