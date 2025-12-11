# 1brc.go
My attempt of the 1 Billion Row Challenge in Go

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