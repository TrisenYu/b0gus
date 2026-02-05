#### Low-interative, Configuration-Directed golang Honeypots server 

Currently, **B0gus** is still under development. B0gus supports to deploy _SSH_, _NTP_ as a functional honeypot. 'Functional' here means that you can assign non-privilege port number, fetch records (etc. `attackers' IP,`, `user name`, `passwords`, `public keys`, `ssh versions`, `accessing time`) from interactions and reload the configuration of ssh service in real time.

In future, B0gus will:
- use docker to further isolate itself from other benign processes;
- deploy itself as an automatic service;
- support utilize root privilege on certain ports;
- integrate with LLM for extending the ability during the whole interaction


#### References

#### License
B0gus is distributed under the terms of the 3-Clauses-BSD License. See the included file `license` in the root direnctory of b0gus for more details.

#### Todo-list

- if possible, set up [oss-fuzz](https://google.github.io/oss-fuzz/getting-started/new-project-guide/)
- other todo mentioned in the source code