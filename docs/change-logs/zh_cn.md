### 版本 $0.1.2$
- 完善文档。
- 改进LLM的接入方式。
- 加入对SSH、NTP与DNS的mock单元测试。
- 加入余额查询接口。
- 修改fake SMTP的实现逻辑。
- 修复`terminal/line_editor_aux`在多行输入文本存在时，光标回退至中部再输入回车所产生的输出渲染错误。
#### 计划
- 拟计划于fake_http中提供伪装远控木马能力。
- 拟计划为HTTP加入页面前端。
- 修复emoji字符对`terminal/line_editor_aux.go`造成的光标错位影响。
- 调研[mvdan.cc/sh/v3/syntax](https://mvdan.cc/sh/v3/syntax)。
- 调研PowerShell内置Parser做windows系统下的命令行语法检查。
- 调用LLM完成特定工作时，限制并发数与额度开销。
