# room-server 文件与函数说明

> 由 `scripts/docs/generate_room_code_map.py` 从源码注释生成。
> 范围：`services/room-server` 下全部非测试 Go 文件；函数说明取自函数上方的一行中文注释。
> 测试文件（`_test.go`）未包含。相关文档见 [api-server 文件与函数说明](api-server-code-map.md)。

## 统计

| 目录 | 文件数 | 函数数 |
| --- | ---: | ---: |
| `cmd/server` | 1 | 2 |
| `internal/app` | 1 | 3 |
| `internal/config` | 1 | 2 |
| `internal/protocol` | 8 | 20 |
| `internal/protocol/snapshot` | 2 | 13 |
| `internal/room` | 6 | 22 |
| `internal/session` | 2 | 10 |
| `internal/transport` | 3 | 11 |
| **合计** | **24** | **83** |

## cmd/server

### `cmd/server/main.go`

功能：room-server 启动入口

| 函数 | 作用 |
| --- | --- |
| `main` | 房间服务启动入口 |
| `run` | 启动房间服务主流程 |

## internal/app

### `internal/app/run.go`

功能：服务依赖组装、监听与生命周期管理

| 函数 | 作用 |
| --- | --- |
| `Run` | 运行房间服务并管理监听生命周期 |
| `unexpectedExit` | 构造非预期退出错误 |
| `healthHandler` | 提供健康检查 HTTP 处理器 |

## internal/config

### `internal/config/config.go`

功能：房间服务配置加载与校验

| 函数 | 作用 |
| --- | --- |
| `Load` | 加载并解析服务配置 |
| `Config.validate` | 校验服务配置 |

## internal/protocol

### `internal/protocol/admission_proof.go`

功能：进房准入凭证解析

| 函数 | 作用 |
| --- | --- |
| `DecodeAdmissionProof` | 解析进房准入凭证 |
| `DecodeJoinAdmission` | DecodeJoinAdmission decodes the join fingerprint and proof from the same frame. |

### `internal/protocol/enter_game.go`

功能：进房玩家列表编码

| 函数 | 作用 |
| --- | --- |
| `MinimumRoomFrameBytes` | 计算进房帧最小字节数 |
| `EncodeEnterGame` | 编码进房玩家列表 |
| `writeRosterPlayer` | 写入单个玩家记录 |

### `internal/protocol/frame.go`

功能：客户端与服务端帧头处理

| 函数 | 作用 |
| --- | --- |
| `decodeClientFrame` | 解析客户端帧 |
| `newServerFrame` | 创建服务端帧缓冲区 |

### `internal/protocol/game_message.go`

功能：游戏消息编解码

| 函数 | 作用 |
| --- | --- |
| `DecodeGameMessage` | 解析游戏消息 |
| `EncodeGameMessage` | 编码游戏消息 |
| `EncodeLevelElection` | 编码关卡选举消息 |

### `internal/protocol/join_game.go`

功能：加入游戏指纹解析

| 函数 | 作用 |
| --- | --- |
| `DecodeJoinGameFingerprint` | 解析加入游戏指纹 |

### `internal/protocol/level_update.go`

功能：关卡更新消息解析

| 函数 | 作用 |
| --- | --- |
| `DecodeLevelUpdate` | 解析关卡更新消息 |

### `internal/protocol/player_notifications.go`

功能：玩家进出与关卡变更通知编解码

| 函数 | 作用 |
| --- | --- |
| `EncodePlayerJoined` | 编码玩家加入通知 |
| `EncodePlayerLeft` | 编码玩家离开通知 |
| `EncodePlayerChangedLevel` | 编码玩家切换关卡通知 |
| `DecodeMoveResult` | 解析移动结果消息 |
| `DecodeCancelMove` | 解析取消移动消息 |
| `decodeMoveControl` | 解析移动控制消息 |

### `internal/protocol/time.go`

功能：时间同步消息编解码

| 函数 | 作用 |
| --- | --- |
| `DecodeTimePing` | 解析时间同步请求 |
| `EncodeTimePong` | 编码时间同步响应 |

## internal/protocol/snapshot

### `internal/protocol/snapshot/delta.go`

功能：快照差量生成与应用

| 函数 | 作用 |
| --- | --- |
| `createDelta` | 生成快照差量 |
| `applyDelta` | 应用快照差量 |

### `internal/protocol/snapshot/snapshot.go`

功能：快照帧解析、读写与历史管理

| 函数 | 作用 |
| --- | --- |
| `Parse` | 解析快照帧 |
| `Frame.Bytes` | 返回快照帧字节 |
| `Reader.LatestSequence` | 返回已读到的最新序号 |
| `Reader.Read` | 读取并应用快照帧 |
| `Writer.Write` | 写入快照数据并返回帧 |
| `Writer.Ack` | 记录对端确认序号 |
| `Writer.discardBefore` | 丢弃指定序号前的历史 |
| `Writer.allocateSequence` | 分配下一个快照序号 |
| `findKeyframe` | 查找可用的关键帧 |
| `remember` | 记录快照关键帧历史 |
| `checksum` | 计算快照校验和 |

## internal/room

### `internal/room/admission.go`

功能：（未提供文件说明）

| 函数 | 作用 |
| --- | --- |
| `Server.begin` | 开始处理进房请求 |
| `Server.complete` | 完成进房校验并加入房间 |

### `internal/room/connections.go`

功能：（未提供文件说明）

| 函数 | 作用 |
| --- | --- |
| `Server.drop` | 丢弃指定连接 |
| `Server.remove` | 移除指定玩家 |

### `internal/room/dispatch.go`

功能：（未提供文件说明）

| 函数 | 作用 |
| --- | --- |
| `Server.receiveActive` | 处理已进房连接的数据 |

### `internal/room/gameplay.go`

功能：房间游戏消息与快照转发

| 函数 | 作用 |
| --- | --- |
| `connection.group` | 返回连接所在关卡分组 |
| `Server.receiveGameMessage` | 处理客户端游戏消息 |
| `Server.receiveSnapshot` | 处理客户端快照消息 |
| `encodePlayerState` | 编码玩家状态快照数据 |
| `Server.sendSnapshot` | 向目标发送快照 |
| `Server.sendGameMessage` | 向目标发送游戏消息 |

### `internal/room/level.go`

功能：关卡分组与玩家关卡切换

| 函数 | 作用 |
| --- | --- |
| `Server.levelFor` | 获取连接所在关卡状态 |
| `Server.changeLevel` | 处理玩家切换关卡 |
| `Server.leaveLevel` | 处理玩家离开关卡 |
| `Server.sendElection` | 发送关卡选举消息 |
| `Server.refreshLevelRoster` | 刷新关卡玩家列表 |

### `internal/room/server.go`

功能：房间服务事件循环与玩家管理

| 函数 | 作用 |
| --- | --- |
| `NewServer` | 创建房间服务实例 |
| `Server.Serve` | 运行房间事件循环 |
| `Server.Close` | 关闭传输资源 |
| `Server.handleEvent` | 分发底层网络事件 |
| `Server.broadcast` | 向其他玩家广播数据 |
| `Server.rosterAndNextID` | 生成玩家列表与下一个玩家 ID |

## internal/session

### `internal/session/admission_verifier.go`

功能：调用 API 校验进房准入

| 函数 | 作用 |
| --- | --- |
| `NewAdmissionVerifier` | 创建进房准入校验器 |
| `AdmissionVerifier.Verify` | 调用 API 校验进房凭证 |

### `internal/session/registry.go`

功能：连接会话与准入状态注册表

| 函数 | 作用 |
| --- | --- |
| `NewRegistry` | 创建会话注册表 |
| `Registry.Connect` | 登记新连接票据 |
| `Registry.StartJoin` | 标记票据进入加入流程 |
| `Registry.Admit` | 标记票据已准入 |
| `Registry.Disconnect` | 断开并移除连接票据 |
| `Registry.Expire` | 清理超时票据 |
| `Registry.Online` | 判断账号是否在线 |
| `Registry.current` | 读取票据当前记录 |

## internal/transport

### `internal/transport/event.go`

功能：传输层事件结构定义

无顶层函数。

### `internal/transport/native_linux.go`

功能：Linux 原生 UDP 传输实现

| 函数 | 作用 |
| --- | --- |
| `OpenNative` | 打开原生 UDP 传输 |
| `Native.Poll` | 轮询传输事件 |
| `Native.Send` | 发送数据到指定连接 |
| `Native.Disconnect` | 断开指定连接 |
| `Native.Close` | 关闭传输资源 |

### `internal/transport/native_windows.go`

功能：Windows 原生传输实现

| 函数 | 作用 |
| --- | --- |
| `OpenNative` | 打开原生 UDP 传输 |
| `openBridge` | 打开 Windows 网桥传输 |
| `Native.Poll` | 轮询传输事件 |
| `Native.Send` | 发送数据到指定连接 |
| `Native.Disconnect` | 断开指定连接 |
| `Native.Close` | 关闭传输资源 |
