#### Low-interative, Configuration-Directed golang Honeypots server 

Currently, **B0gus** only supports to deploy _SSH_ as a functional honeypot by utilizing library `golang/x/crypto/ssh` and database ORM framework `gorm`. 'Functional' here means that you can assign non-privilege port number, fetch records (etc. `attackers' IP,`, `user name`, `passwords`, `public keys`, `ssh versions`, `accessing time`) from interactions and reload the configuration of ssh service in real time.

In future, B0gus will:
- use docker to further isolate itself from other benign processes;
- deploy itself as an automatic service;
- support utilize root privilege on certain ports;
- integrate with LLM for extending the ability during the whole interaction


#### References

#### Todo-list

- if possible, set up [oss-fuzz](https://google.github.io/oss-fuzz/getting-started/new-project-guide/)