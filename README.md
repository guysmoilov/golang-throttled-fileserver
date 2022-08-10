# Tiny, throttled HTTP DVC remote

This implements a very simple HTTP fileserver and valid DVC remote in Go.
Uploads are throttled by default to 30KB per second, configurable via the `BYTES_PER_SEC` env var.

## Motivation - reliably reproduce corrupted DVC file uploads

**The symptom: DVC push uploads a seemingly random chunk of some of the files it pushes, and reports success at the end of the upload.**

The uploaded chunk(s) seems to come from the middle of the file, skipping its start.

User remote caches then become corrupted in a way that's hard to recover from, and data may be lost permanently. 

The problem seems to happen when the following conditions are met:
* Slow network
* Several large-ish files being pushed (though 1 is often enough)
* Only HTTP remote was tested, but maybe others are also affected

So this server and procedure was written to emulate that in a clean environment, so that DVC can be debugged.

## Recreating the DVC push issue
This was tested on DVC 2.11.0

1. Run the Go server with `go run .` (go version 1.18)
2. Clone the following repo and `dvc pull` its data:
https://dagshub.com/nirbarazida/hello-world
3. Add the Go remote to the clone's list of remote `dvc remote add local http://localhost:3030`
4. Push the data `dvc push -r local`
5. You'll see that the upload is very slow, 30KB per second per file - simulating real world problematic conditions
6. Even before the upload is done, you should be able to see the issue:
   1. Compare the start of one of the uploading data files - most consistently, the data file which fails is `enron.csv`
      with md5 `994e368ae5a22a6db1943a397e7c4308`
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
   5. If not, check the other uploaded files as well.
1. Optionally, confirm the problem with Wireshark - it can be clearly seen by following the HTTP stream of one of the broken files.