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
Unfortunately function closures cannot be inspected in any way.
*/
package memsize
