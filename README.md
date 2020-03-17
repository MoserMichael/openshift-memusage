

## Motivation for this investigation

Currently the resource requirements of openshift are rather large - as compared to other solutions based on kubernetes; 
a basic openshift cluster may use up to 64gb of RAM, I think that this might turn out to be a big problem for Red Hat in the long term, this due to the following reasons:
I think that a low barrier of entry is an important reason for the success of an enterprise system as a whole; I think that the current success of Red Hat is an example for this thesis:

some twenty years ago a lot of people were trying out Linux at home, and found that there was a lot to gain from using this operating system
in the course of these experiments they built up a collective expertise that helped to establish Linux (and RHEL by extension) as a widely accepted enterprise solution

Compare that to the current state:
- currently kubernetes is in very high demand, but a high resource requirements of openshift will make it hard for the average person to run this system at home - on his personal equipment.
- As a result there will be fewer persons with a deep knowledge of the openshift platform, the circle of users who could use this knowledge at the workplace will be smaller in the next generation of techies.


By comparison: i think that a low barrier of entry is purposely built into the AWS cloud environment;
- it is relatively easy to come up with a working solution based on AWS solutions, but it will be very expensive to operate that solution.
- However the financial result will be to the benefit of Mr. Besos.


An idea for a possible direction of inquiry:
We might be saving some memory with the control plane by running a set of operators/operator loops as part of the same process, this would have the potential for a lower memory footprint as there are fewer instances of garbage collected heaps that are otherwise maintained on a per process basis. 


## Theory

I suspect we are loosing a lot due to internal fragmentation of the heap. In more detail: memory allocators like the one that is being used by the gloang runtime are typically using a big number of fixed sized pools - each fixed sized pool instance is providing memory allocations of a nearest size class;  With this approach you end up with a lot of memory pages having been reserved in advance. This project aims to first measure the amount of internal fragmentation of an openshift installation and then come up with recomendations on how improve the situation.


## Plan of action

- Learning about mem. alloc in golang.
    - tcmalloc (no longer used, but it's well documented and is the grandaddy of them)
    - current mem alloc package in golang runtime. (src/runtime/malloc.go in https://github.com/golang/go )
    - what can we get out of ReadMemStats api in terms of data?
- Quantify the internal fragmentation
    - decide on goal:
        - measure for each operator
        - measure for a representative subset of operators.
            - decide on what that subset is.
    - alloc in golang has a statistic feature: API ReadMemStats / returns MemStats
        - can add library that listens on named piped and returns MemStats report
    - can this library be injected into golang process address space? (or do we rebuild the operator for that)
        - looked at some projects i worked with; they all produce ELF executables with dependencies on glibc;
            - injection possible: if we create a shared library wrapper for glibc that forwards all function,
              except for adding reporting func in a  common entrypoint.
- study approaches how to reduce mem consumption in openshift
    - tuning of mem allocator so that less is cached per instance. (probably at perf. cost)
    - combining separate operators into the same project.
        - one operator running in each os thread? (virtual core)
        - problem: they all use the same client-go classes
            - client go uses RESTClient as transport mechanism. RESTCLient can be tuned for concurrent access, is that enough?
     - utilizing operator-sdk
        - each project will get the option to build the operator standalone, or as a loadable library.
        - loadable library: can create a host operator that loads multiple operator libraries, so that each operator library will run in its own virtual core/thread.
