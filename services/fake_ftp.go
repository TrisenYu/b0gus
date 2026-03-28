package services

// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD

/*
References:
	https://www.cs.colostate.edu/helpdocs/ftp.html
	https://github.com/DearRude/easyGoFTP/blob/master/ftpserver/ftpserver.go

	One way to emulate a ftp server is to create a very easy virtual memory file system
	Port number: 20 and 21
	20: data path
	21: command path
	The appearance of ftp shell is like:

	ftp> ls
	229 Entering Extended Passive Mode (|||44893|)
	150 Here comes the directory listing.
	drwxrwxrwx    2 1000     1000         4096 Sep 27 15:26 Desktop
	drwxr-xr-x    9 1000     1000         4096 Nov 25 15:16 Zotero
	drwxrwxr-x   14 1000     1000         4096 Nov 25 15:09 apps
	ftp> ?
	Commands may be abbreviated.  Commands are:

		!               close           fget            lpage           modtime         pdir            rcvbuf          sendport        type
		$               cr              form            lpwd            more            pls             recv            set             umask
		account         debug           ftp             ls              mput            pmlsd           reget           site            unset
		append          delete          gate            macdef          mreget          preserve        remopts         size            usage
		ascii           dir             get             mdelete         msend           progress        rename          sndbuf          user
		bell            disconnect      glob            mdir            newer           prompt          reset           status          verbose
		binary          edit            hash            mget            nlist           proxy           restart         struct          xferbuf
		bye             epsv            help            mkdir           nmap            put             rhelp           sunique         ?
		case            epsv4           idle            mls             ntrans          pwd             rmdir           system
		cd              epsv6           image           mlsd            open            quit            rstatus         tenex
		cdup            exit            lcd             mlst            page            quote           runique         throttle
		chmod           features        less            mode            passive         rate            send            trace

	Compared to low-interaction ssh honeypot, it will be much more laborious to implement ftp shell
*/

// uncomfortable to use memory file system
// also feel tried at the interactive shell provided for human/remote nasty programs
// TODO: use sandbox/container to handle files-related operations
// accept file but strictly strip it privilege and save it in a sandbox.
