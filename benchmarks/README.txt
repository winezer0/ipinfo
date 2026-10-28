XDB 查询基线与 Searcher 池评估

准备资源：python scripts/fetch_test_assets.py
运行：go run ./cmd/ipbench -xdb assets/ip2region_v4.xdb -requests 10000 -out benchmarks/ipv4-run1.json
IPv6：替换为 assets/ip2region_v6.xdb。
默认组合：pool=1,4；workers=1,4,16；batch=1,16。
每种地址族至少运行三次，串行运行，避免与下载、race、其他压测同时执行。

pool=1 是当前独占查询语义；pool=4 是候选池，使用四个独立文件句柄和同一份 VectorIndex。
pool=1 直接调用现有实例，不增加 channel 借还；pool>1 才使用 channel 租借。
池通过有界 channel 租借查询器，每个查询器保留原有互斥锁，结束时关闭全部句柄。
该候选仅用于评估，不改变生产构造器、API 配置或默认资源占用。

JSON 包含请求/秒、IP 查询/秒、请求 P50/P95、分配字节/次数及近似池堆内存。
Windows 使用 QueryPerformanceCounter，其他平台使用 Go 单调时钟。
单次延迟包含等待池的时间，批量延迟是整个批次耗时。
输入 IP 可能重复；BatchFindFull 仍逐个执行查询，但结果按 IP 去重。
测量为闭环负载，使用预热后的 OS 文件缓存，不含 HTTP、ASN、网络、初始化时间和进程 RSS。
P95 是本次采样的分位数，不代表开放式线上负载的尾延迟。
只有真实业务压测也稳定受益时，才考虑把候选池加入生产路径。
