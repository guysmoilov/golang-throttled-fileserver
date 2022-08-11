# Tiny, intermittently failing HTTP DVC remote

This implements a very simple HTTP fileserver and valid DVC remote in Go.
Uploads fail on the first attempt, to cause the DVC client to retry the upload - which exposes a bug described below.

## Motivation - reliably reproduce corrupted DVC file uploads

**The symptom: DVC push uploads a seemingly random chunk of some of the files it pushes, and reports success at the end of the upload.**

The uploaded chunk(s) seems to come from the middle of the file, skipping its start.  
Short files appear to be truncated completely on the retry, resulting in empty files.

This happens in the wild, when pushing to HTTP/S and Webdav remotes, with a slightly sketchy connections or when pushing
large-is files.

See the [attached wireshark capture file](diagnostics/corrupted-push-example.pcap) for a recreation of the issue.  
To easily see the 2 requests, the first valid one one which fails, and the second corrupted one which succeeds,
Then after opening wireshark click Statistics -> Conversations, which will show 2 TCP streams, one for each request.  
To see the HTTP content, then click on one of the TCP conversation and then the "Follow stream" button.
You can see that the first short conversation stops with a 500 error from the server (intentionally), and then
the second long conversation succeeds but the data sent to the server is corrupted.

User remote caches then become corrupted in a way that's hard to recover from, and data may be lost permanently. 

## Recreating the DVC push issue
This was tested on DVC 2.11.0 and 2.17.0

1. Run the Go server with `go run .` (go version 1.18)
2. Clone the following repo and `dvc pull` its data:
https://dagshub.com/nirbarazida/hello-world
3. Add the Go remote to the clone's list of remotes `dvc remote add local http://localhost:3030`
4. Push the data `dvc push -r local`
5. You'll see that DVC reports a successful push
6. To see the issue:
   1. Wait until the `dvc push` ends (will probably report failing on some of the files)
   1. Compare the start of one of the uploading data files - e.g. `enron.csv` with md5 `994e368ae5a22a6db1943a397e7c4308`
   2. Run `head content/99/4e368ae5a22a6db1943a397e7c4308` inside the Go server dir to see what was uploaded
   3. Compare to the correct `head data/enron.csv` in the cloned hello-world repo
      * It should look like this:
        ```
        filename,label,text
        enron1/ham/1061.2000-05-10.farmer.ham.txt,ham,"Subject: ena sales on hpl
        just to update you on this project ' s status :
        based on a new report that scott mills ran for me from sitara , i have come up
        with the following counterparties as the ones to which ena is selling gas off
        of hpl ' s pipe .
        altrade transaction , l . l . c . gulf gas utilities company
        brazoria , city of panther pipeline , inc .
        central illinois light company praxair , inc .
        central power and light company reliant energy - entex
        ```
   4. You should see a totally different start to the file in `head content/99/4e368ae5a22a6db1943a397e7c4308`
   5. If not, check the other uploaded files as well. The small ones get pushed as empty files.
1. Optionally, confirm the problem with Wireshark - it can be clearly seen by following the HTTP stream of one of the broken files.
1. Observation on what happens:
   *  It seems that the DVC client starts by POSTing the correct data, but then after the initial request
      fails, it retries the POST with a different chunk of data - and since the web server is stateless,
      it just accepts this chunk as a valid HTTP request and dutifully saves the partial data
   * **I suspect the problem is in or around `aiohttp_retry.RetryClient` - it seems retries don't seek back to the start of the input file?**
      * Seems more complicated than that, since the sent chunks in the retries don't seem to be consecutive -
        there are gaps between them.
      