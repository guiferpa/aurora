goos: darwin
goarch: arm64
pkg: github.com/guiferpa/aurora/lexer
cpu: Apple M3
BenchmarkScanToken_Keyword-8         	78417979	        15.25 ns/op	       0 B/op	       0 allocs/op
BenchmarkScanToken_Identifier-8      	73019718	        17.32 ns/op	       0 B/op	       0 allocs/op
BenchmarkScanToken_Number-8          	131816068	         9.118 ns/op	       0 B/op	       0 allocs/op
BenchmarkScanToken_HexNumber-8       	100000000	        10.64 ns/op	       0 B/op	       0 allocs/op
BenchmarkScanToken_String-8          	151201761	         7.927 ns/op	       0 B/op	       0 allocs/op
BenchmarkScanToken_SingleChar-8      	192291577	         7.268 ns/op	       0 B/op	       0 allocs/op
BenchmarkScanToken_Comment-8         	167061290	         7.180 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetTokens_Simple-8          	 3057578	       444.7 ns/op	    1360 B/op	      14 allocs/op
BenchmarkGetTokens_Medium-8          	 1000000	      1201 ns/op	    3984 B/op	      37 allocs/op
BenchmarkGetTokens_Complex-8         	  349312	      4002 ns/op	   13488 B/op	     102 allocs/op
BenchmarkGetTokens_Hex-8             	  981447	      1231 ns/op	    3984 B/op	      37 allocs/op
BenchmarkGetTokens_Strings-8         	 1000000	      1116 ns/op	    3696 B/op	      34 allocs/op
BenchmarkGetTokens_ManyTokens-8      	  346030	      3432 ns/op	   13104 B/op	      98 allocs/op
BenchmarkGetFilledTokens_Complex-8   	  223297	      5263 ns/op	   15648 B/op	     109 allocs/op
BenchmarkIsIdentChar-8               	145176751	         8.246 ns/op	       0 B/op	       0 allocs/op
BenchmarkKeywordsLookup-8            	27211629	        50.22 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/guiferpa/aurora/lexer	25.273s
