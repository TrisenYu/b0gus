所有go可执行文件都附带额外数据段，存放大量供go运行时使用的实用信息。

go的程序计数器行号表(Program Counter Line Number Table, PCLNTAB)是golang编译器生成的一种符号表，
它记录了函数的PC值和源代码行号之间的映射关系。具体而言，其包含以下内容：

- 偏移量表：记录了每个函数在程序二进制文件中的偏移量，即函数的起始地址。
- 行号表：记录了每个函数中每条指令的源代码行号。
- 文件名表：记录了每个文件的文件名和路径。
- 函数名表：记录了每个函数的函数名和包名。

从而，调试器即可根据PCLNTAB中的信息，精确地定位到go源代码中的某一行，方便程序员的调试工作。

除了windows环境下的PE编译产物，通过strip去除了符号表的elf程序，通常依然保留这一数据段。

一方面，这有利于线上生产程序进行符号解析和栈回溯分析。另一方面，这也便于逆向分析恢复函数符号以及分析软件依赖。

![](../../../_future-feats/anti-dbg/imgs/gopclntab.png)

## 编码规则
gopclntab内部结构体与字段默认使用目标机器原生字节序。
go直接通过结构体指针映射内存读取数据，无需额外序列化/反序列化。
所有整型数据遵循程序架构原生大小端序，指针长度匹配目标平台指针位数。

`.gopclntab`头部固定占 $7$ 字节，自go1.2起含义不变，跨平台编码一致。
因此可通过魔数(Go 1.16之前的MagicNumber是0xffffffb，之后是0xfffffffa)字节序判断程序大小端，
后续字段偏移与排布均由魔数版本决定。
所有uintptr类型字段均为相对于`.gopclntab`段起始的偏移量。

ELF 文件中的firstmoduledata 总是在 .noptrdata 这个 Section 里，
PE 文件中可能会在 .data 或 .noptrdata Section，
而 MachO 文件的 firstmoduledata 在 __noptrdata Section 中，
我们可以按照uintptr 为单元遍历section，
检查每个uintptr 指向的地址前四个字节是否为pclntab 的 Magic Number 0xFFFFFFFB 或 0xfffffffa

### 参考
1. [github.com/open-telemetry/opentelemetry-ebpf-profiler/blob/main/doc/gopclntab.md](https://github.com/open-telemetry/opentelemetry-ebpf-profiler/blob/main/doc/gopclntab.md)
