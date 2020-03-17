
# Introduction

the go memory allocator is part of the go runtime; the code of the runtime is [here](https://github.com/golang/go)

The main entry point of the allocator is [mallocgc](https://github.com/golang/go/blob/master/src/runtime/malloc.go) 

Note that in this document I am linking to the source file that contains a function, i am not linking to the line that this function curently is placed on - as this location may well change in the future, so you might have to scroll a bit each time you follow a link.

# main data structures of the go runtime

One needs to know the role of the main classes, in order to understand the flow of the allocator, these are explained [here](https://github.com/golang/go/blob/master/src/runtime/HACKING.md)

- type m - represents an OS thread.
- type p - holds resources and rights required to execute. this means it holds scheduler and memory allocator state.
- type g - holds the code of a go routine. also has the go routines user stack.

These types are defined in file  [/src/runtime/runtime2.go](https://github.com/golang/go/blob/master/src/runtime/runtime2.go)

"The scheduler's job is to match up a G (the code to execute), an M (where to execute it), and a P (the rights and resources to execute it). 
When an M stops executing user Go code, for example by entering a system call, it returns its P to the idle P pool. 
In order to resume executing user Go code, for example on return from a system call, it must acquire a P from the idle pool."


# memory allocator overview

The memory allocator uses a set of fixed sized pools, each of these pools can return a memory block of constant size.
The memory allocator has to service an allocation request of arbitrary size, if first searches for the fixed size pool of the nearest size larger then that of the requested memory block, once such a fixed size pool has been found it is asked to service an allocation.

some important data structures:

- type [mspan](https://github.com/golang/go/blob/master/src/runtime/mheap.go) This object acts as an instance of a fixed size memory pool, it can allocate and free memory blocks that are of the same size.
- type [mcache](https://github.com/golang/go/blob/master/src/runtime/mcache.go) used to allocate memory objects of up to to 32768 bytes (maxSmallSize constant. this object owns a set mspan objecs. There is one mcache per operating system thread - the M type that holds all per thread data owns an mcache; this increases performance because no lock has to be acquired if an allocation is coming from the per thread mcache. Initially all mspan objects are empty, they are filled up upon demand, as allocations are made.


# flow of a memory allocation in go

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
For maintaining the next batch of 64 entries, the mspan has 64 bit integer structure member span.allocCache that is treated as a bitmap; if the 10th bit is zero that means that the 10th element can be allocated. To make the lookup of the first free bit shorter they maintain the next free index span span.freeIndex and shift the bitmap to the left on each call.

(didn't understand why they need a bitmap if they are not freeing any elements)

[mcache nextFree](https://github.com/golang/go/blob/master/src/runtime/malloc.go) - we get here if the current batch of elements covered by bitmap span.allocCache didn't show any free elements, so span.allocCache is refilled - in [refillAllocCache](https://github.com/golang/go/blob/master/src/runtime/malloc.go) the next eight bytes are fetched from span.allocBits and negated (so that zero means a free slot).`[nextFreeIndex](https://github.com/golang/go/blob/master/src/runtime/malloc.go) persists until a free index slot has been found.

if the current span has been exhausted, then it needs to get another memory page for it from this mcache instance (one up in the hierarchy)

[mcache refill](https://github.com/golang/go/blob/master/src/runtime/mcache.go) the current span of the given size class is returned to the heap and we get a new span object from the heap that will stand in for the current size class. Note that this is an expensive operation as a lock needs to be obtained for that purpose in [mcache cacheSpan](https://github.com/golang/go/blob/master/src/runtime/mcentral.go) 

