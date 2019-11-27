# mpc
Secure Multi-Party Computation

# Syntax

## Mathematical operations

    func main(a, b int32) (int32, int32) {
        q, r := a / b
        return q, r
    }

# Issues

 - 1-bit multiplication does not work since z[1] is unwired. Fix this when circuit constants are implemented.
