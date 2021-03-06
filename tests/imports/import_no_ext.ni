import "std/test"

test.run("Import with no extension", fn(assert) {
    import '../../testdata/math' as math
    assert.isTrue(isFunc(math.add))
    assert.isEq(math.add(2, 4), 6)
})
