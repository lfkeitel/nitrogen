import "std/test"

test.run("Import directory", fn(assert) {
    import '../../testdata/math2' as math
    assert.isTrue(isFunc(math.add))
    assert.isEq(math.add(2, 4), 6)
})
