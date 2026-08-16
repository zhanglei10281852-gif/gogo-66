# Bug Reproduction

## 包的性质

当前 test_model_fix 保存的是被测模型修复后的结果源码，不是初始含 Bug 源码。要复现原始缺陷，必须检出下面固定的 parent SHA；不要在当前修复结果源码上期待重新出现修复前失败。生成系统使用的可信验证补丁和完整验证日志仅在本地留存，不提交到结果分支。

## 问题现象

定位给出的不确定窗口把测量点排除在外了。一条 100 km、slack 1.05 的单段系统，A 端 OTDR 量到 21.0 km 电缆距离（换算 KP 20.0），B 端量到 78.75 km（换算 KP 25.0），两条记录都显式写了 uncertainty_km=0.4。定位结果 best_kp=22.5，flags 里确实报了 end_disagreement，但 window_low_kp=21.934、window_high_kp=23.066，window_width_km 只有 1.132，uncertainty_km 只报 0.566；两条 estimates 自己的 route_kp（20.0 和 25.0）全都掉在这个窗口外面，两个 end_estimate_*_kp 也在窗口外。把 B 端改成一致的 84.0 km（同样指向 KP 20）时，窗口宽度还是 1.131，也就是说不管两端差 0 km 还是差 5 km，窗口宽度都只跟着合成 sigma 走。照这个窗口派船，作业段根本不可能包含故障点，下游按 window_low/high 取防护区、埋深和备缆也全是偏小的。请修复不确定窗口的宽度计算，让窗口覆盖最差的单条测量偏差，同时保持证据一致时窗口不被无谓放大、min_window_km/max_window_km 的下限与截断、end_disagreement 与 outlier 标记、best_kp 的加权口径不变，并保证全量测试通过。

## 含 Bug 版本

- 仓库：zhanglei10281852-gif/gogo-66
- 仓库地址：https://github.com/zhanglei10281852-gif/gogo-66.git
- parent SHA：7624152b9ba05b276d9ad662fd58aad6a409f7c7

## 复现步骤

```bash
git clone -- https://github.com/zhanglei10281852-gif/gogo-66.git bug-repro
cd bug-repro
git checkout --detach 7624152b9ba05b276d9ad662fd58aad6a409f7c7
go test ./internal/locate -run "^TestLocalizeWindowBracketsEveryMeasurement$" -count=1 -v
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/locate -run "^TestLocalizeWindowBracketsEveryMeasurement$" -count=1 -v
=== RUN   TestLocalizeWindowBracketsEveryMeasurement
    window_coverage_regression_test.go:72: estimate EV-A at KP 20 is outside the reported window 21.934..23.066
--- FAIL: TestLocalizeWindowBracketsEveryMeasurement (0.00s)
FAIL
FAIL	CableMend/internal/locate	0.002s
FAIL

```

stderr：

```text
warning: internal/locate/window_coverage_regression_test.go has type 100755, expected 100644
warning: internal/locate/window_coverage_regression_test.go has type 100755, expected 100644

```

### linux/arm64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/locate -run "^TestLocalizeWindowBracketsEveryMeasurement$" -count=1 -v
=== RUN   TestLocalizeWindowBracketsEveryMeasurement
    window_coverage_regression_test.go:72: estimate EV-A at KP 20 is outside the reported window 21.934..23.066
--- FAIL: TestLocalizeWindowBracketsEveryMeasurement (0.02s)
FAIL
FAIL	CableMend/internal/locate	0.140s
FAIL

```

stderr：

```text
warning: internal/locate/window_coverage_regression_test.go has type 100755, expected 100644
warning: internal/locate/window_coverage_regression_test.go has type 100755, expected 100644

```

## 通过条件

A 端 21.0 km、B 端 78.75 km（各自 uncertainty_km=0.4）时：每条 estimate 的 route_kp 与两个 end_estimate_*_kp 都落在 [window_low_kp, window_high_kp] 内，window_width_km 不小于 4.9，uncertainty_km 不小于 2.4；两条一致证据（都指向 KP 20.0）时 window_width_km 仍不超过 1.5 且不小于 localization.min_window_km；clamped 测量、单端证据、outlier/end_disagreement 标记、LocalizeAll 分组与输入顺序无关性等既有行为不回归；定向测试、全量 go test ./... -count=1 与 go build ./... && go vet ./... 全部通过；校准与远端复跑均在 golang:1.22 linux/amd64 单架构完成。
