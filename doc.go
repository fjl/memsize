/*
Package memsize computes the size of your object graph.

So you made a spiffy algorithm and it works really well, but geez it's using
way too much memory. Where did it all go? memsize to the rescue!

To get started, create a RootSet and add some roots:

    var rs memsize.RootSet
    rs.Add("my object", obj)
    rs.Add("some other object", obj2)

You can traverse the graph to get all the objects and their respective sizes:

    sizes := rs.Scan()
    fmt.Println(sizes.Total())

memsize can handle cycles just fine and tracks both private and public struct fields.
There are a few limitations though:

Races during scan

Scanning your objects while they are being modified is racy. There is no solution
to this at the moment, but you can work around it by synchronizing your goroutines
around the call to Scan.

Channel buffers cannot be scanned

There is no way to get the content of a channel buffer without reading from the channel.
memsize will report size based on the cap of the channel, but can't tell you how much
memory is used its elements.
*/
package memsize
