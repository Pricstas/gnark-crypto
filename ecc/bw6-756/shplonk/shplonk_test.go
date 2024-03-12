// Copyright 2020 Consensys Software Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by consensys/gnark-crypto DO NOT EDIT

package shplonk

import (
	"crypto/sha256"
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bw6-756/fr"
	"github.com/consensys/gnark-crypto/ecc/bw6-756/kzg"
	"github.com/stretchr/testify/require"
)

// Test SRS re-used across tests of the KZG scheme
var testSrs *kzg.SRS
var bAlpha *big.Int

func init() {
	const srsSize = 230
	bAlpha = new(big.Int).SetInt64(42) // randomise ?
	testSrs, _ = kzg.NewSRS(ecc.NextPowerOfTwo(srsSize), bAlpha)
}

func TestOpening(t *testing.T) {

	assert := require.New(t)

	nbPolys := 2
	sizePoly := make([]int, nbPolys)
	for i := 0; i < nbPolys; i++ {
		sizePoly[i] = 10 + i
	}
	polys := make([][]fr.Element, nbPolys)
	for i := 0; i < nbPolys; i++ {
		polys[i] = make([]fr.Element, sizePoly[i])
		for j := 0; j < sizePoly[i]; j++ {
			polys[i][j].SetRandom()
		}
	}

	digests := make([]kzg.Digest, nbPolys)
	for i := 0; i < nbPolys; i++ {
		digests[i], _ = kzg.Commit(polys[i], testSrs.Pk)
	}

	points := make([][]fr.Element, nbPolys)
	for i := 0; i < nbPolys; i++ {
		points[i] = make([]fr.Element, i+2)
		for j := 0; j < i+2; j++ {
			points[i][j].SetRandom()
		}
	}

	hf := sha256.New()

	// correct proof
	openingProof, err := BatchOpen(polys, digests, points, hf, testSrs.Pk)
	assert.NoError(err)
	err = BatchVerify(openingProof, digests, points, hf, testSrs.Vk)
	assert.NoError(err)

	// tampered proof
	openingProof.ClaimedValues[0][0].SetRandom()
	err = BatchVerify(openingProof, digests, points, hf, testSrs.Vk)
	assert.Error(err)

}

func TestBuildZtMinusSi(t *testing.T) {

	nbSi := 10
	points := make([][]fr.Element, nbSi)
	sizeSi := make([]int, nbSi)
	nbPoints := 0
	for i := 0; i < nbSi; i++ {
		sizeSi[i] = 5 + i
		nbPoints += sizeSi[i]
		points[i] = make([]fr.Element, sizeSi[i])
		for j := 0; j < sizeSi[i]; j++ {
			points[i][j].SetRandom()
		}
	}
	for i := 0; i < nbSi; i++ {
		ztMinusSi := buildZtMinusSi(points, i)
		if len(ztMinusSi) != nbPoints-sizeSi[i]+1 {
			t.Fatal("deg(Z_{T-S_{i}}) should be nbPoints-size(S_{i})")
		}
		for j := 0; j < nbSi; j++ {
			if j == i {
				for k := 0; k < sizeSi[j]; k++ {
					y := eval(ztMinusSi, points[j][k])
					if y.IsZero() {
						t.Fatal("Z_{T-S_{i}}(S_{i}) should not be zero")
					}
				}
				continue
			}
			for k := 0; k < sizeSi[j]; k++ {
				y := eval(ztMinusSi, points[j][k])
				if !y.IsZero() {
					t.Fatal("Z_{T-S_{i}}(S_{j}) should be zero")
				}
			}
		}
	}

}

func TestInterpolate(t *testing.T) {

	nbPoints := 10
	x := make([]fr.Element, nbPoints)
	y := make([]fr.Element, nbPoints)
	for i := 0; i < nbPoints; i++ {
		x[i].SetRandom()
		y[i].SetRandom()
	}
	f := interpolate(x, y)
	for i := 0; i < nbPoints; i++ {
		fx := eval(f, x[i])
		if !fx.Equal(&y[i]) {
			t.Fatal("f(x_{i})!=y_{i}")
		}
	}

}

func TestBuildLagrangeFromDomain(t *testing.T) {

	nbPoints := 10
	points := make([]fr.Element, nbPoints)
	for i := 0; i < nbPoints; i++ {
		points[i].SetRandom()
	}
	var r fr.Element
	for i := 0; i < nbPoints; i++ {

		l := buildLagrangeFromDomain(points, i)

		// check that l(xᵢ)=1 and l(xⱼ)=0 for j!=i
		for j := 0; j < nbPoints; j++ {
			y := eval(l, points[j])
			if i == j {
				if !y.IsOne() {
					t.Fatal("l_{i}(x_{i}) should be equal to 1")
				}
			} else {
				if !y.IsZero() {
					t.Fatal("l_{i}(x_{j}) where i!=j should be equal to 0")
				}
			}
		}
		r.SetRandom()
		y := eval(l, r)
		if y.IsZero() {
			t.Fatal("l_{i}(x) should not be zero if x is random")
		}
	}

}

func TestBuildVanishingPoly(t *testing.T) {
	s := 10
	x := make([]fr.Element, s)
	for i := 0; i < s; i++ {
		x[i].SetRandom()
	}
	r := buildVanishingPoly(x)

	if len(r) != s+1 {
		t.Fatal("error degree r")
	}

	// check that r(xᵢ)=0 for all i
	for i := 0; i < len(x); i++ {
		y := eval(r, x[i])
		if !y.IsZero() {
			t.Fatal("πᵢ(X-xᵢ) at xᵢ should be zero")
		}
	}

	// check that r(y)!=0 for a random point
	var a fr.Element
	a.SetRandom()
	y := eval(r, a)
	if y.IsZero() {
		t.Fatal("πᵢ(X-xᵢ) at r \neq xᵢ should not be zero")
	}
}

func TestMultiplyLinearFactor(t *testing.T) {

	s := 10
	f := make([]fr.Element, s, s+1)
	for i := 0; i < 10; i++ {
		f[i].SetRandom()
	}

	var a, y fr.Element
	a.SetRandom()
	f = multiplyLinearFactor(f, a)
	y = eval(f, a)
	if !y.IsZero() {
		t.Fatal("(X-a)f(X) should be zero at a")
	}
	a.SetRandom()
	y = eval(f, a)
	if y.IsZero() {
		t.Fatal("(X-1)f(X) at a random point should not be zero")
	}

}

func TestNaiveMul(t *testing.T) {

	size := 10
	f := make([]fr.Element, size)
	for i := 0; i < size; i++ {
		f[i].SetRandom()
	}

	nbPoints := 10
	points := make([]fr.Element, nbPoints)
	for i := 0; i < nbPoints; i++ {
		points[i].SetRandom()
	}

	v := buildVanishingPoly(points)
	buf := make([]fr.Element, size+nbPoints-1)
	g := mul(f, v, buf)

	// check that g(xᵢ) = 0
	for i := 0; i < nbPoints; i++ {
		y := eval(g, points[i])
		if !y.IsZero() {
			t.Fatal("f(X)(X-x_{1})..(X-x_{n}) at x_{i} should be zero")
		}
	}

	// check that g(r) != 0 for a random point
	var a fr.Element
	a.SetRandom()
	y := eval(g, a)
	if y.IsZero() {
		t.Fatal("f(X)(X-x_{1})..(X-x_{n}) at a random point should not be zero")
	}

}

func TestDiv(t *testing.T) {

	nbPoints := 10
	s := 10
	f := make([]fr.Element, s, s+nbPoints)
	for i := 0; i < s; i++ {
		f[i].SetRandom()
	}

	// backup
	g := make([]fr.Element, s)
	copy(g, f)

	// successive divions of linear terms
	x := make([]fr.Element, nbPoints)
	for i := 0; i < nbPoints; i++ {
		x[i].SetRandom()
		f = multiplyLinearFactor(f, x[i])
	}
	q := make([][2]fr.Element, nbPoints)
	for i := 0; i < nbPoints; i++ {
		q[i][1].SetOne()
		q[i][0].Neg(&x[i])
		f = div(f, q[i][:])
	}

	// g should be equal to f
	if len(f) != len(g) {
		t.Fatal("lengths don't match")
	}
	for i := 0; i < len(g); i++ {
		if !f[i].Equal(&g[i]) {
			t.Fatal("f(x)(x-a)/(x-a) should be equal to f(x)")
		}
	}

	// division by a degree > 1 polynomial
	for i := 0; i < nbPoints; i++ {
		x[i].SetRandom()
		f = multiplyLinearFactor(f, x[i])
	}
	r := buildVanishingPoly(x)
	f = div(f, r)

	// g should be equal to f
	if len(f) != len(g) {
		t.Fatal("lengths don't match")
	}
	for i := 0; i < len(g); i++ {
		if !f[i].Equal(&g[i]) {
			t.Fatal("f(x)(x-a)/(x-a) should be equal to f(x)")
		}
	}

}
