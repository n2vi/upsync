# upsync
[`upspinfs`](https://github.com/upspin/upspin/blob/master/cmd/upspinfs/doc.go) on Mac and Linux is great, but is not well supported on Windows or BSD.  Although the `upspin` command works cross-platform, it is tedious to use for more than occasional file transfers.  The new command `upsync` aims to bridge the gap, helping keep a local disk directory tree in sync with a master version in the Upspin file system.

To start, create a local directory whose path ends in a string that looks like an existing upspin directory, for example on BSD you can `mkdir ~/u/grosse@gmail.com/Public` then `cd` there and execute `upsync.`  Make local edits to the downloaded files or create new files, and then `upsync` to upload your changes to the Upspin master. To discard your local changes, just remove the edited local files and `upsync.`  (Executing both local `rm` and `upspin rm` are required to remove content permanently.)

There are no command flags or config files or environment variables.  Performance will be best if `cacheserver` is already running.  To minimize surprise, upsync prints which files it is uploading or downloading and declines to download files larger than 50MB.  It promises never to write outside the starting directory and subdirectories and, as an initial way to enforce that, declines all symlinks.

There are no clever merge heuristics;  copying back and forth proceeds by a trivial "newest wins" rule.  This requires some discipline in remembering to `upsync` after each editing session and is better suited to single person rather than joint editing.  Don't let your computer clocks drift.

Eventually, I hope to solidify FUSE support on Windows and OpenBSD and switch there to the much preferable `upspinfs.`  But even then `upsync` may have some niche benefits:
* enables work offline, i.e. a workaround for the distributed `upspinfs` we have not yet built
* offers mitigation of user misfortune, for example when they discard their upspin keys
* provides a worked out example for new Upspin client developers
* leaves a backup in case cloud store or Upspin projects die without warning

### Windows checklist
This tool was written assuming you are an experienced Upspin user trying to assist a friend with file sharing or backup.  Here is a checklist if your friend is on Windows 10:
1. create or check existing upspin account and permissions  _It is helpful if you can provide them space on an existing server._
1. confirm `\Users\x\upspin\config` is correct
1. disk must be NTFS (because FAT has peculiar timestamps)
1. open a powershell window
1. install `go` and `git,` if not already there
1. `go get -u upspin.io/cmd/...`
1. fetch `upsync.go; go install`   _Be aware that Go files must be transferred as UTF8, else expect a NUL compile warning._
1. `mkdir \Users\x\u`...
1. `upsync`
