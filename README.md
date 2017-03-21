gocut is an alternative to `cut` which only trims leading and trailing bytes.

It was created because `cut -b` was too slow when processing large streams of
data.

Usage
-----

```
Usage of gocut:
  -cpuprofile string
        Write CPU profile to disk
  -leadingbytes int
        Number of leading bytes to trim
  -outfile string
        Location to save log lines
  -trailingbytes int
        Number of trailing bytes to trim
```

You can specify `-` as the `outfile` to print to stdout.
