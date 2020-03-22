package main

import (
	"fmt"
	"runtime"
    "math"
)

func dumpMemStats(stat *runtime.MemStats)  {

    if stat.Alloc != 0 {
        fmt.Printf("Alloc:          %30d [bytes of allocated heap objects]\n",  stat.Alloc)
    }
    if stat.Alloc != 0 {
        fmt.Printf("TotalAlloc      %30d [cumulative bytes allocated for heap objects]\n",  stat.TotalAlloc)
    }
    if stat.Sys != 0 {
        fmt.Printf("Sys:            %30d [total bytes of memory obtained from the OS]\n",  stat.Sys)
    }
    if stat.Lookups != 0 {
        fmt.Printf("Lookups:        %30d\n",stat.Lookups)
    }
    if stat.Mallocs != 0 {
        fmt.Printf("Mallocs:        %30d [cumulative count of heap objects allocated]\n",stat.Mallocs)
    }
    if stat.Frees != 0 {
        fmt.Printf("Frees:          %30d [cumulative count of heap objects freed]\n",  stat.Frees)
    }
    if stat.HeapAlloc != 0 {
	    fmt.Printf("HeapAlloc:      %30d [bytes of allocated heap objects]\n", stat.HeapAlloc)
    }
    if stat.HeapSys != 0 {
	    fmt.Printf("HeapSys:        %30d [bytes of heap memory obtained from the OS (including reserved)]\n",  stat.HeapSys)
    }

    /*
	ratio := float64(stat.HeapSys) / float64(stat.HeapAlloc)
	fmt.Printf("Ratio (HeapSys/HeapAlloc) %g\n", ratio)
    */

    if stat.HeapIdle != 0 {
        var val int64
        if stat.HeapIdle > math.MaxInt64 {
            val = int64(math.MaxUint64 - stat.HeapIdle)
            val = - val - 1
        } else {
            val = int64(stat.HeapIdle)
        }

        fmt.Printf("HeapIdle:       %30d [bytes in idle (unused) spans]\n", val)
    }
    if stat.HeapInuse != 0 {
        fmt.Printf("HeapInuse:      %30d [bytes in in-use spans]\n", stat.HeapInuse)
    }
    if stat.HeapReleased != 0 {
        fmt.Printf("HeapReleased:   %30d [bytes of physical memory returned to the OS]\n", stat.HeapReleased)
    }
    if stat.HeapObjects != 0 {
        fmt.Printf("HeapObjects:    %30d [number of allocated heap objects]\n", stat.HeapObjects)
    }
    if stat.StackInuse != 0 {
        fmt.Printf("StackInuse:     %30d [bytes in stack spans] \n",  stat.StackInuse)
    }
    if stat.StackSys != 0 {
        fmt.Printf("StackSys:       %30d [bytes of stack memory obtained from the OS]\n",    stat.StackSys)
    }
    if stat.MSpanInuse != 0 {
        fmt.Printf("MSpanInuse:     %30d [bytes of allocated mspan structures]\n",  stat.MSpanInuse)
    }
    if stat.MSpanSys != 0 {
        fmt.Printf("MSpanSys:       %30d [bytes of memory obtained from the OS for mspan]\n",    stat.MSpanSys)
    }
    if stat.MCacheInuse != 0 {
        fmt.Printf("MCacheInuse:    %30d [of allocated mcache structure]\n", stat.MCacheInuse)
    }
    if stat.MCacheSys != 0 {
        fmt.Printf("MCacheSys:      %30d [bytes of memory obtained from the OS for mcache structures]\n",   stat.MCacheSys)
    }
    if stat.BuckHashSys != 0 {
        fmt.Printf("BuckHashSys:    %30d [bytes of memory in profiling bucket hash tables]\n", stat.BuckHashSys)
    }
    if stat.GCSys != 0 {
        fmt.Printf("GCSys:          %30d [memory in garbage collection metadata]\n",       stat.GCSys)
    }
    if stat.OtherSys != 0 {
        fmt.Printf("OtherSys:       %30d [memory in miscellaneous off-heap runtime allocations]\n",    stat.OtherSys)
    }
    if stat.NextGC != 0 {
        fmt.Printf("NextGC:         %30d [target heap size of the next GC cycle]\n",      stat.NextGC)
    }
    if stat.LastGC != 0 {
        fmt.Printf("LastGC:         %30d [time the last garbage collection finished]\n",      stat.LastGC)
    }
    if stat.PauseTotalNs != 0 {
        fmt.Printf("PauseTotalNs:   %30d [cumulative nanoseconds in GC]\n",stat.PauseTotalNs)
    }
    if stat.NumGC != 0 {
        fmt.Printf("NumGC:          %30d [number of completed GC cycles]\n",       stat.NumGC)
    }
    if stat.NumForcedGC != 0 {
        fmt.Printf("NumForcedGC:    %30d [number of GC cycles that were forced by the application calling the GC function]\n", stat.NumForcedGC)
    }

    var totalMalloc uint64
    var totalFree  uint64
    var nonEmpty int

	for i := 0; i < 61; i += 1 {
		if stat.BySize[i].Mallocs != 0 || stat.BySize[i].Frees != 0 {
			fmt.Printf("sizeClass: %d Size: %d Mallocs %d Frees %d\n", i, stat.BySize[i].Size, stat.BySize[i].Mallocs, stat.BySize[i].Frees)
            nonEmpty += 1
            totalMalloc += stat.BySize[i].Mallocs
            totalFree += stat.BySize[i].Frees
		}
	}

    if  nonEmpty > 1 {
        fmt.Printf("\nIn all non empty size classes: malloc calls: %d free calls: %d", totalMalloc, totalFree)
    }
}

func dumpEmptySizeClasses(stat *runtime.MemStats)  {

    fmt.Printf("\n\nEmpty size classes:\n\n")


	for i := 0; i < 61; i += 1 {
		if stat.BySize[i].Mallocs == 0 && stat.BySize[i].Frees == 0 {
			fmt.Printf("sizeClass: %d Size: %d Mallocs %d Frees %d\n", i, stat.BySize[i].Size, stat.BySize[i].Mallocs, stat.BySize[i].Frees)
		}
	}
}



func getStats() *runtime.MemStats {

	stat := new(runtime.MemStats)
	runtime.ReadMemStats(stat)

	return stat
}

func diffMemStats(statB *runtime.MemStats, stat *runtime.MemStats) (*runtime.MemStats) {

    var stat2 runtime.MemStats

    stat2 = *statB

    stat2.Alloc         -= stat.Alloc
    stat2.TotalAlloc    -= stat.TotalAlloc
    stat2.Sys           -= stat.Sys
    stat2.Lookups       -= stat.Lookups
    stat2.Mallocs       -= stat.Mallocs
    stat2.Frees         -= stat.Frees
	stat2.HeapAlloc     -= stat.HeapAlloc
	stat2.HeapSys       -= stat.HeapSys

    stat2.HeapIdle      -= stat.HeapIdle

    stat2.HeapInuse     -= stat.HeapInuse
    stat2.HeapReleased  -= stat.HeapReleased
    stat2.HeapObjects   -= stat.HeapObjects
    stat2.StackInuse    -= stat.StackInuse
    stat2.StackSys      -= stat.StackSys
    stat2.MSpanInuse    -= stat.MSpanInuse
    stat2.MSpanSys      -= stat.MSpanSys
    stat2.MCacheInuse   -= stat.MCacheInuse
    stat2.MCacheSys     -= stat.MCacheSys
    stat2.BuckHashSys   -= stat.BuckHashSys
    stat2.GCSys         -= stat.GCSys
    stat2.OtherSys      -= stat.OtherSys
    stat2.NextGC        -= stat.NextGC
    stat2.LastGC        -= stat.LastGC
    stat2.PauseTotalNs  -= stat.PauseTotalNs
    stat2.NumGC         -= stat.NumGC
    stat2.NumForcedGC   -= stat.NumForcedGC

    for i := 0; i < 61; i += 1 {
		stat2.BySize[i].Mallocs -= stat.BySize[i].Mallocs
		stat2.BySize[i].Frees -= stat.BySize[i].Frees
	}

    return &stat2
}

func main() {

	st1 := getStats()

    numberVar := new(int32)

    st2 := getStats()

    tarr := make([]byte,1024)

	st3 := getStats()

    tarr2 := make([]byte,20000)

	st4 := getStats()

	fmt.Printf("\nBaseline \n\n")
	dumpMemStats(st1)
    dumpEmptySizeClasses(st1)

	fmt.Printf("\nDiff (alloc int32) \n\n")
    std := diffMemStats(st2, st1)
	dumpMemStats(std)

	fmt.Printf("\nDiff (alloc make([]byte,1024) \n\n")
    std = diffMemStats(st3, st2)
	dumpMemStats(std)


	fmt.Printf("\nDiff (alloc make([]byte,20000) \n\n")
    std = diffMemStats(st4, st3)
	dumpMemStats(std)


    fmt.Printf("\n\nptr of allocations %p %p %p\n", numberVar, tarr, tarr2)


}
