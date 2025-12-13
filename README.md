# 1brc.go
My attempt for the [1 Billion Row Challenge](https://github.com/gunnarmorling/1brc) in Go

## 1st Iteration
- Single thread, naive approach
```
________________________________________________________
Executed in   73.42 secs    fish           external
   usr time   71.72 secs    0.00 micros   71.72 secs
   sys time    2.92 secs  476.00 micros    2.92 secs
```

## 2nd Iteration
- Change `scanner.Text()` to `scanner.Bytes()`
- Replace `ParseFloat()` with manual parsing
```
________________________________________________________
Executed in   62.57 secs    fish           external
   usr time   60.67 secs    0.00 micros   60.67 secs
   sys time    2.63 secs  434.00 micros    2.63 secs
```

## 3rd Iteration
- Divide file into chunks and process concurrently with goroutines
```
________________________________________________________
Executed in   36.73 secs    fish           external
   usr time   69.42 secs  427.00 micros   69.42 secs
   sys time   12.06 secs    0.00 micros   12.06 secs
```

![Profile result for 3rd Iteration](doc/profile_3.png)
At this point I was stuck on how to further optimize and turned to profiling. The results indicated that `mapaccess2_faststr` was the primary bottleneck (this is the function for map lookups with the `comma, ok` check), followed by memory allocation `mallocgc`. With no obvious optimization strategy coming to mind, I decided to procrastinate and tackle the third candidate first, `mapassign_faststr`.

In the current implementation, the map stores `Data` structs by value. Every time the map needs to be updated, the code retrieves the existing value from the map, constructs a new `Data` struct and writes it back into the map. This means that every update results in a map assignment. To reduce the cost of these repeated assignments, I changed the map to store pointers to `Data` instead. In this way, assignment only happens once when a key is inserted, subsequent updates mutate the `Data` struct in place via the pointer.

## 4th Iteration
- Store pointers to `Data` struct as map value
```
________________________________________________________
Executed in   34.28 secs    fish           external
   usr time   53.75 secs  346.00 micros   53.75 secs
   sys time   11.85 secs    0.00 micros   11.85 secs
```
![Profile result for 4th Iteration](doc/profile_4.png)
As a result, `mapassign_faststr` effectively disappeared in the graph. The improvement in execution time is marginal, since the optimization basically reduces the number of map writes at the cost of additional pointer dereferencing.
