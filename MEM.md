
# Introduction

the go memory allocator is part of the go runtime; the code of the runtime is [here](https://github.com/golang/go)

The main entry point of the allocator is [mallocgc](https://github.com/golang/go/blob/master/src/runtime/malloc.go) 

Note that in this document I am linking to the source file that contains a function, i am not linking to the line that this function curently is placed on - as this location may well change in the future, so you might have to scroll a bit each time you follow a link.

# Main data structures of the go runtime

One needs to know the role of the main classes, in order to understand the flow of the allocator, these are explained [here](https://github.com/golang/go/blob/master/src/runtime/HACKING.md)

- type m - represents an OS thread.
- type p - holds resources and rights required to execute. this means it holds scheduler and memory allocator state.
- type g - holds the code of a go routine. also has the go routines user stack.

These types are defined in file  [/src/runtime/runtime2.go](https://github.com/golang/go/blob/master/src/runtime/runtime2.go)

"The scheduler's job is to match up a G (the code to execute), an M (where to execute it), and a P (the rights and resources to execute it). 
When an M stops executing user Go code, for example by entering a system call, it returns its P to the idle P pool. 
In order to resume executing user Go code, for example on return from a system call, it must acquire a P from the idle pool."

They runtime also puts data maintained per OS thread into the M object - this per thread data can be accessed without the need of obtaining a lock.

# Memory allocator overview

The memory allocator uses a set of fixed sized pools, each of these pools can return a memory block of constant size.
The memory allocator has to service an allocation request of arbitrary size, if first searches for the fixed size pool of the nearest size larger then that of the requested memory block, once such a fixed size pool has been found it is asked to service an allocation.

some important data structures:

- type [mspan](https://github.com/golang/go/blob/master/src/runtime/mheap.go) This object acts as an instance of a fixed size memory pool, it can allocate and free memory blocks that are of the same size.
- type [mcache](https://github.com/golang/go/blob/master/src/runtime/mcache.go) used to allocate memory objects of up to to 32768 bytes (maxSmallSize constant. this object owns a set mspan objecs. There is one mcache per operating system thread - the M type that holds all per thread data owns an mcache; this increases performance because no lock has to be acquired if an allocation is coming from the per thread mcache. Initially all mspan objects are empty, they are filled up upon demand, as allocations are made.


# Flow of a memory allocation in go

the allocator entry point is function [mallocgc](https://github.com/golang/go/blob/master/src/runtime/malloc.go) Actually the task of mallocgc is relatively simple as there is no 'free' function in a garbage collected language ;-)

```
    // Allocate an object of size bytes.
    // Small objects are allocated from the per-P cache's free lists.
    // Large objects (> 32 kB) are allocated straight from the heap.
    func mallocgc(size uintptr, typ *_type, needzero bool) unsafe.Pointer {
```

as parameter we get the size of the allocation, needzero indicates if the newly allocated memory block needs to be filled with zeros. typ - indicates if the memory is for an object that does not contains pointers or not; for each size class it maintains two fixed sized allocators (span objects) one for objects that contain pointers, and another one for plain old data objects that don't have pointers - these are treated differently by the garbage collector scan.


flow of mallocgc

## start of function

- check if garbage collector allows to proceed with allocation now:
    - per go routine structure G has a signed integer counter gcAssistBytes, for each allocation this counter is decremented by the size of the allocation. the allocation is allowed to proceed if the counter remains positive; it it drops below zero then some garbage collection scans may need to be done in gcAssistAlloc, as a result this counter is incremented  (gcAssistAlloc is done inf file [/src/runtime/mgcmark.go](https://github.com/golang/go/blob/master/src/runtime/mgcmark.go) 

- obtain the m structure that represents an OS thread; [acquirem](https://github.com/golang/go/blob/master/src/runtime/runtime1.go) gets it and increments the lock counter in m; after that they set the mallocing flag in m, to prevent recursions (if recursion is detected they abort)


## allocation

- obtain the mcache object for allocations up to 32768 bytes (maxSmallSize constant); If m the os thread data object has it, get it from here, otherwise there is a global mcache object that is used.

- 'tiny allocations' of size up to 16 bytes (up to maxTinySize constant)

- 'small allocations' in size range 16 ... 32768 bytes (maxTinySize..maxSmallSize constants)
    - get the size class. the size class is the array index used for lookup of the the fixed size allocator instance. For sizes up to smallSizeMax(1024) there is a fixed size allocator in size steps of smallSizeDiv(8); for range of 1024..3276 there is one in steps of 128 bytes (largeSizeDiv)
    - once the span object has been found it is used and a lookup of a currently free entry is nextFreeFast ; if no free entry is available then the span object is refilled in nextFree
            
[mspan nextFreeFast](https://github.com/golang/go/blob/master/src/runtime/malloc.go) - if a free object is available in the mspan object then this function returns a pointer to it. 

Each span object owns a a memory page (span.base) that can be thought of as an array of elements of the same size (span.elemsize)
For maintaining the next batch of 64 entries, the mspan has an array of 64 bit integer structure member span.allocCache that is treated as a bitmap; if the 10th bit is zero that means that the 10th element can be allocated. To make the lookup of the first free bit shorter they maintain the next free index span span.freeIndex and shift the bitmap to the left on each call.

Of course ther is no explicit function for freeing a object that is part of the bitmap, however the object might become unreachabel (i.e. no other object points at it); in this case it will be cleaned out by a garbage collection pass.

[mcache nextFree](https://github.com/golang/go/blob/master/src/runtime/malloc.go) - we get here if the current batch of elements covered by bitmap span.allocCache didn't show any free elements, so span.allocCache is refilled - in [refillAllocCache](https://github.com/golang/go/blob/master/src/runtime/malloc.go) the next eight bytes are fetched from span.allocBits and negated (so that zero means a free slot).`[nextFreeIndex](https://github.com/golang/go/blob/master/src/runtime/malloc.go) persists until a free index slot has been found.

if the current span has been exhausted, then it needs to get another memory page for it from this mcache instance (one up in the hierarchy)

[mcache refill](https://github.com/golang/go/blob/master/src/runtime/mcache.go) the current span of the given size class is returned to the heap and we get a new span object from the heap that will stand in for the current size class. Note that this is an expensive operation as a lock needs to be obtained for that purpose in [mcache cacheSpan](https://github.com/golang/go/blob/master/src/runtime/mcentral.go) 


## allocation statistics

The function [ReadMemStats](https://github.com/golang/go/blob/master/src/runtime/mstats.go)  copies memory statistics into structure MemStats.

Here they use a peculiar trick: all memory statistics are kept in a struct of type mstats that has exactly the same layout as structure MemStats. The difference is 
that all fields in MemStats are visible from other packages (the field names start with a capital letter) while the fields in structure mstats are private (start with a lower case letter).
The internal copy function readmemstats_m then copies two seemingly unrelated structures by mean of the following statement. don't do this at home!

```
	memmove(unsafe.Pointer(stats), unsafe.Pointer(&memstats), sizeof_C_MStats)
```

call ReadMemStats is quite expensive: in order to keep us from seing an outdated copy of the memory statistics all other threads must be stopped at garbage collection save points, then after copying the structure to the user supplied MemStats arguments all the threads are started up again. (so this function shouldn't be called frequently)

Lets write a small test program in order to learn about memory statistics. 

The program first calls ReadMemStat and prints the counters as a baseline.

Then it does three allocations, for each of these allocation it shows the difference of the values returned by ReadMemStat - relative to the values we had before the allocation.
We have three allocations

- a very small one of four bytes length in a size class that already had seen allocations before
- then a larger one - an array of 1024 bytes (from a size class that has no prior allocations); 
- then a very large one - 20000 bytes (this on isn't covered by any size class - it's a large allocation)

Some things that can be seen from this program run on go 1.13.6: (output of test program at the end of this file)

- a hello world go program preallocates about 63M of memory and puts it into unused span object (MSpanSys)
- by the time we hit the main function there are already 21 non empty size classes (out of 60) we had 178 allocation before hitting main (as displayed in the Baseline)


## test program output

Here is the output of the test program on go 1.13.6

```
Baseline 

Alloc:                                  119688 [bytes of allocated heap objects]
TotalAlloc                              119688 [cumulative bytes allocated for heap objects]
Sys:                                  69928960 [total bytes of memory obtained from the OS]
Mallocs:                                   179 [cumulative count of heap objects allocated]
Frees:                                       1 [cumulative count of heap objects freed]
HeapAlloc:                              119688 [bytes of allocated heap objects]
HeapSys:                              66813952 [bytes of heap memory obtained from the OS (including reserved)]
HeapIdle:                             66387968 [bytes in idle (unused) spans]
HeapInuse:                              425984 [bytes in in-use spans]
HeapReleased:                         66322432 [bytes of physical memory returned to the OS]
HeapObjects:                               178 [number of allocated heap objects]
StackInuse:                             294912 [bytes in stack spans] 
StackSys:                               294912 [bytes of stack memory obtained from the OS]
MSpanInuse:                               6664 [bytes of allocated mspan structures]
MSpanSys:                                16384 [bytes of memory obtained from the OS for mspan]
MCacheInuse:                             13888 [of allocated mcache structure]
MCacheSys:                               16384 [bytes of memory obtained from the OS for mcache structures]
BuckHashSys:                              2212 [bytes of memory in profiling bucket hash tables]
GCSys:                                 2240512 [memory in garbage collection metadata]
OtherSys:                               544604 [memory in miscellaneous off-heap runtime allocations]
NextGC:                                4473924 [target heap size of the next GC cycle]
sizeClass: 1 Size: 8 Mallocs 5 Frees 0
sizeClass: 2 Size: 16 Mallocs 51 Frees 0
sizeClass: 3 Size: 32 Mallocs 32 Frees 0
sizeClass: 4 Size: 48 Mallocs 17 Frees 0
sizeClass: 5 Size: 64 Mallocs 5 Frees 0
sizeClass: 6 Size: 80 Mallocs 2 Frees 0
sizeClass: 7 Size: 96 Mallocs 7 Frees 0
sizeClass: 8 Size: 112 Mallocs 1 Frees 0
sizeClass: 9 Size: 128 Mallocs 4 Frees 0
sizeClass: 14 Size: 208 Mallocs 9 Frees 0
sizeClass: 17 Size: 256 Mallocs 1 Frees 0
sizeClass: 19 Size: 320 Mallocs 1 Frees 0
sizeClass: 21 Size: 384 Mallocs 14 Frees 0
sizeClass: 22 Size: 416 Mallocs 3 Frees 0
sizeClass: 24 Size: 480 Mallocs 1 Frees 0
sizeClass: 28 Size: 704 Mallocs 1 Frees 0
sizeClass: 30 Size: 896 Mallocs 7 Frees 0
sizeClass: 32 Size: 1152 Mallocs 3 Frees 0
sizeClass: 36 Size: 1792 Mallocs 4 Frees 0
sizeClass: 43 Size: 4096 Mallocs 1 Frees 0
sizeClass: 50 Size: 8192 Mallocs 1 Frees 0
sizeClass: 51 Size: 9472 Mallocs 8 Frees 0

In all non empty size classes: malloc calls: 178 free calls: 0

Empty size classes:

sizeClass: 0 Size: 0 Mallocs 0 Frees 0
sizeClass: 10 Size: 144 Mallocs 0 Frees 0
sizeClass: 11 Size: 160 Mallocs 0 Frees 0
sizeClass: 12 Size: 176 Mallocs 0 Frees 0
sizeClass: 13 Size: 192 Mallocs 0 Frees 0
sizeClass: 15 Size: 224 Mallocs 0 Frees 0
sizeClass: 16 Size: 240 Mallocs 0 Frees 0
sizeClass: 18 Size: 288 Mallocs 0 Frees 0
sizeClass: 20 Size: 352 Mallocs 0 Frees 0
sizeClass: 23 Size: 448 Mallocs 0 Frees 0
sizeClass: 25 Size: 512 Mallocs 0 Frees 0
sizeClass: 26 Size: 576 Mallocs 0 Frees 0
sizeClass: 27 Size: 640 Mallocs 0 Frees 0
sizeClass: 29 Size: 768 Mallocs 0 Frees 0
sizeClass: 31 Size: 1024 Mallocs 0 Frees 0
sizeClass: 33 Size: 1280 Mallocs 0 Frees 0
sizeClass: 34 Size: 1408 Mallocs 0 Frees 0
sizeClass: 35 Size: 1536 Mallocs 0 Frees 0
sizeClass: 37 Size: 2048 Mallocs 0 Frees 0
sizeClass: 38 Size: 2304 Mallocs 0 Frees 0
sizeClass: 39 Size: 2688 Mallocs 0 Frees 0
sizeClass: 40 Size: 3072 Mallocs 0 Frees 0
sizeClass: 41 Size: 3200 Mallocs 0 Frees 0
sizeClass: 42 Size: 3456 Mallocs 0 Frees 0
sizeClass: 44 Size: 4864 Mallocs 0 Frees 0
sizeClass: 45 Size: 5376 Mallocs 0 Frees 0
sizeClass: 46 Size: 6144 Mallocs 0 Frees 0
sizeClass: 47 Size: 6528 Mallocs 0 Frees 0
sizeClass: 48 Size: 6784 Mallocs 0 Frees 0
sizeClass: 49 Size: 6912 Mallocs 0 Frees 0
sizeClass: 52 Size: 9728 Mallocs 0 Frees 0
sizeClass: 53 Size: 10240 Mallocs 0 Frees 0
sizeClass: 54 Size: 10880 Mallocs 0 Frees 0
sizeClass: 55 Size: 12288 Mallocs 0 Frees 0
sizeClass: 56 Size: 13568 Mallocs 0 Frees 0
sizeClass: 57 Size: 14336 Mallocs 0 Frees 0
sizeClass: 58 Size: 16384 Mallocs 0 Frees 0
sizeClass: 59 Size: 18432 Mallocs 0 Frees 0
sizeClass: 60 Size: 19072 Mallocs 0 Frees 0

Diff (alloc int32) 

Alloc:                                      16 [bytes of allocated heap objects]
TotalAlloc                                  16 [cumulative bytes allocated for heap objects]
Mallocs:                                     1 [cumulative count of heap objects allocated]
HeapAlloc:                                  16 [bytes of allocated heap objects]
HeapObjects:                                 1 [number of allocated heap objects]
sizeClass: 2 Size: 16 Mallocs 1 Frees 0

Diff (alloc make([]byte,1024) 

Alloc:                                    1024 [bytes of allocated heap objects]
TotalAlloc                                1024 [cumulative bytes allocated for heap objects]
Mallocs:                                     1 [cumulative count of heap objects allocated]
HeapAlloc:                                1024 [bytes of allocated heap objects]
HeapIdle:                                -8192 [bytes in idle (unused) spans]
HeapInuse:                                8192 [bytes in in-use spans]
HeapObjects:                                 1 [number of allocated heap objects]
MSpanInuse:                                136 [bytes of allocated mspan structures]
sizeClass: 31 Size: 1024 Mallocs 1 Frees 0

Diff (alloc make([]byte,20000) 

Alloc:                                   20480 [bytes of allocated heap objects]
TotalAlloc                               20480 [cumulative bytes allocated for heap objects]
Mallocs:                                     1 [cumulative count of heap objects allocated]
HeapAlloc:                               20480 [bytes of allocated heap objects]
HeapIdle:                               -40960 [bytes in idle (unused) spans]
HeapInuse:                               40960 [bytes in in-use spans]
HeapObjects:                                 1 [number of allocated heap objects]
MSpanInuse:                                272 [bytes of allocated mspan structures]


```

