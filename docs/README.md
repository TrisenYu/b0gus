
See introductions of b0gus which have been translated into different languages below.

- [English](readmes/readme-en.md)
- [中文](readmes/readme-zh_cn.md)

## Low-interactive, Configuration-Directed golang Honeypot

Currently, **bogus** is still under active development stage. Some functions are not yet fully developed.
b0gus supports to deploy _SSH_, _HTTP_, _SMTP_, and _NTP_ as a functional honeypot.
'Functional' here means that you can assign non-privilege port number,
fetch records (etc. `attackers' IP,`, `user name`, `passwords`, `public keys`, `ssh versions`, `accessing time`, `attacking pattern`)
from interactions and reload the configuration of ssh service in real time.

In the future, b0gus will:
- Subsequent additions will include automated configuration for port traffic forwarding.
- further isolation of honeypots using containers;
- support for distributed configuration pushing;
- introduce LLM to enhance camouflage and interactive capabilities.
- collect APIs of various applications to establish active traceability and correlation analysis capabilities.

### License
b0gus is distributed under the terms of the 3-Clauses-BSD License. See the included file `license` in the root directory of b0gus for more details.

### Todo-list

- if possible, set up [oss-fuzz](https://google.github.io/oss-fuzz/getting-started/new-project-guide/)
- other todos mentioned in the source code
- attempt to run on Windows

### References
mentioned in code.
