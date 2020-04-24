package ftoa

import (
	"math"
	"math/big"
	"strconv"
)

const (
	exp_11     = 0x3ff00000
	frac_mask1 = 0xfffff
	bletch     = 0x10
	quick_max  = 14
	int_max    = 14
)

type FToStrMode int

const (
	ModeStandard            FToStrMode = iota /* Either fixed or exponential format; round-trip */
	ModeStandardExponential                   /* Always exponential format; round-trip */
	ModeFixed                                 /* Round to <precision> digits after the decimal point; exponential if number is large */
	ModeExponential                           /* Always exponential format; <precision> significant digits */
	ModePrecision                             /* Either fixed or exponential format; <precision> significant digits */
)

var (
	tens = [...]float64{
		1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9,
		1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19,
		1e20, 1e21, 1e22,
	}

	bigtens = [...]float64{1e16, 1e32, 1e64, 1e128, 1e256}

	big5  = big.NewInt(5)
	big10 = big.NewInt(10)

	dtoaModes = []int{
		ModeStandard:            0,
		ModeStandardExponential: 0,
		ModeFixed:               3,
		ModeExponential:         2,
		ModePrecision:           2,
	}
)

/*
mode:
		0 ==> shortest string that yields d when read in
			and rounded to nearest.
		1 ==> like 0, but with Steele & White stopping rule;
			e.g. with IEEE P754 arithmetic , mode 0 gives
			1e23 whereas mode 1 gives 9.999999999999999e22.
		2 ==> max(1,ndigits) significant digits.  This gives a
			return value similar to that of ecvt, except
			that trailing zeros are suppressed.
		3 ==> through ndigits past the decimal point.  This
			gives a return value similar to that from fcvt,
			except that trailing zeros are suppressed, and
			ndigits can be negative.
		4,5 ==> similar to 2 and 3, respectively, but (in
			round-nearest mode) with the tests of mode 0 to
			possibly return a shorter string that rounds to d.
			With IEEE arithmetic and compilation with
			-DHonor_FLT_ROUNDS, modes 4 and 5 behave the same
			as modes 2 and 3 when FLT_ROUNDS != 1.
		6-9 ==> Debugging modes similar to mode - 4:  don't try
			fast floating-point estimate (if applicable).

		Values of mode other than 0-9 are treated as mode 0.
*/
func ftoa(d float64, mode int, biasUp bool, ndigits int, buf []byte) ([]byte, int) {
	var sign bool
	if math.IsNaN(d) {
		buf = append(buf, "NaN"...)
		return buf, 9999
	}
	if math.Signbit(d) {
		sign = true
		d = math.Copysign(d, 1.0)
	} else {
		sign = false
	}
	if math.IsInf(d, 0) {
		if sign {
			buf = append(buf, '-')
		}
		buf = append(buf, "Infinity"...)
		return buf, 9999
	}
	dBits := math.Float64bits(d)
	word0 := uint32(dBits >> 32)
	word1 := uint32(dBits)
	if d == 0 {
		// no_digits
		buf = append(buf, '0')
		return buf, 1
	}
	if sign {
		buf = append(buf, '-')
	}
	b, be, bbits := d2b(d)
	i := int((word0 >> exp_shift1) & (exp_mask >> exp_shift1))
	var d2 float64
	var denorm bool
	if i != 0 {
		d2 = setWord0(d, (word0&frac_mask1)|exp_11)
		i -= bias
		denorm = false
	} else {
		/* d is denormalized */
		i = bbits + be + (bias + (p - 1) - 1)
		var x uint64
		if i > 32 {
			x = uint64(word0)<<(64-i) | uint64(word1)>>(i-32)
		} else {
			x = uint64(word1) << (32 - i)
		}
		d2 = setWord0(float64(x), uint32((x>>32)-31*exp_mask))
		i -= (bias + (p - 1) - 1) + 1
		denorm = true
	}
	/* At this point d = f*2^i, where 1 <= f < 2.  d2 is an approximation of f. */
	ds := (d2-1.5)*0.289529654602168 + 0.1760912590558 + float64(i)*0.301029995663981
	k := int(ds)
	if ds < 0.0 && ds != float64(k) {
		k-- /* want k = floor(ds) */
	}
	k_check := true
	if k >= 0 && k < len(tens) {
		if d < tens[k] {
			k--
		}
		k_check = false
	}
	/* At this point floor(log10(d)) <= k <= floor(log10(d))+1.
	   If k_check is zero, we're guaranteed that k = floor(log10(d)). */
	j := bbits - i - 1
	var b2, s2, b5, s5 int
	/* At this point d = b/2^j, where b is an odd integer. */
	if j >= 0 {
		b2 = 0
		s2 = j
	} else {
		b2 = -j
		s2 = 0
	}
	if k >= 0 {
		b5 = 0
		s5 = k
		s2 += k
	} else {
		b2 -= k
		b5 = -k
		s5 = 0
	}
	/* At this point d/10^k = (b * 2^b2 * 5^b5) / (2^s2 * 5^s5), where b is an odd integer,
	   b2 >= 0, b5 >= 0, s2 >= 0, and s5 >= 0. */
	if mode < 0 || mode > 9 {
		mode = 0
	}
	try_quick := true
	if mode > 5 {
		mode -= 4
		try_quick = false
	}
	leftright := true
	var ilim, ilim1 int
	switch mode {
	case 0, 1:
		ilim, ilim1 = -1, -1
		ndigits = 0
	case 2:
		leftright = false
		fallthrough
	case 4:
		if ndigits <= 0 {
			ndigits = 1
		}
		ilim, ilim1 = ndigits, ndigits
	case 3:
		leftright = false
		fallthrough
	case 5:
		i = ndigits + k + 1
		ilim = i
		ilim1 = i - 1
	}
	/* ilim is the maximum number of significant digits we want, based on k and ndigits. */
	/* ilim1 is the maximum number of significant digits we want, based on k and ndigits,
	   when it turns out that k was computed too high by one. */
	fast_failed := false
	if ilim >= 0 && ilim <= quick_max && try_quick {

		/* Try to get by with floating-point arithmetic. */

		i = 0
		d2 = d
		k0 := k
		ilim0 := ilim
		ieps := 2 /* conservative */
		/* Divide d by 10^k, keeping track of the roundoff error and avoiding overflows. */
		if k > 0 {
			ds = tens[k&0xf]
			j = k >> 4
			if (j & bletch) != 0 {
				/* prevent overflows */
				j &= bletch - 1
				d /= bigtens[len(bigtens)-1]
				ieps++
			}
			for ; j != 0; i++ {
				if (j & 1) != 0 {
					ieps++
					ds *= bigtens[i]
				}
				j >>= 1
			}
			d /= ds
		} else if j1 := -k; j1 != 0 {
			d *= tens[j1&0xf]
			for j = j1 >> 4; j != 0; i++ {
				if (j & 1) != 0 {
					ieps++
					d *= bigtens[i]
				}
				j >>= 1
			}
		}
		/* Check that k was computed correctly. */
		if k_check && d < 1.0 && ilim > 0 {
			if ilim1 <= 0 {
				fast_failed = true
			} else {
				ilim = ilim1
				k--
				d *= 10.
				ieps++
			}
		}
		/* eps bounds the cumulative error. */
		eps := float64(ieps)*d + 7.0
		eps = setWord0(eps, _word0(eps)-(p-1)*exp_msk1)
		if ilim == 0 {
			d -= 5.0
			if d > eps {
				buf = append(buf, '1')
				k++
				return buf, int(k + 1)
			}
			if d < -eps {
				buf = append(buf, '0')
				return buf, 1
			}
			fast_failed = true
		}
		if !fast_failed {
			fast_failed = true
			if leftright {
				/* Use Steele & White method of only
				 * generating digits needed.
				 */
				eps = 0.5/tens[ilim-1] - eps
				for i = 0; ; {
					l := int64(d)
					d -= float64(l)
					buf = append(buf, byte('0'+l))
					if d < eps {
						return buf, k + 1
					}
					if 1.0-d < eps {
						buf, k = bumpUp(buf, k)
						return buf, k + 1
					}
					i++
					if i >= ilim {
						break
					}
					eps *= 10.0
					d *= 10.0
				}
			} else {
				/* Generate ilim digits, then fix them up. */
				eps *= tens[ilim-1]
				for i = 1; ; i++ {
					l := int64(d)
					d -= float64(l)
					buf = append(buf, byte('0'+l))
					if i == ilim {
						if d > 0.5+eps {
							buf, k = bumpUp(buf, k)
							return buf, k + 1
						} else if d < 0.5-eps {
							buf = stripTrailingZeroes(buf)
							return buf, k + 1
						}
						break
					}
					d *= 10.0
				}
			}
		}
		if fast_failed {
			if sign {
				buf = buf[:1]
			} else {
				buf = buf[:0]
			}
			d = d2
			k = k0
			ilim = ilim0
		}
	}

	/* Do we have a "small" integer? */
	if be >= 0 && k <= int_max {
		/* Yes. */
		ds = tens[k]
		if ndigits < 0 && ilim <= 0 {
			if ilim < 0 || d < 5*ds || (!biasUp && d == 5*ds) {
				if sign {
					buf = buf[:1]
				} else {
					buf = buf[:0]
				}
				buf = append(buf, '0')
				return buf, 1
			}
			buf = append(buf, '1')
			k++
			return buf, k + 1
		}
		for i = 1; ; i++ {
			l := int64(d / ds)
			d -= float64(l) * ds
			buf = append(buf, byte('0'+l))
			if i == ilim {
				d += d
				if (d > ds) || (d == ds && (((l & 1) != 0) || biasUp)) {
					buf, k = bumpUp(buf, k)
				}
				break
			}
			d *= 10.0
			if d == 0 {
				break
			}
		}
		return buf, k + 1
	}

	m2 := b2
	m5 := b5
	var mhi, mlo *big.Int
	if leftright {
		if mode < 2 {
			if denorm {
				i = be + (bias + (p - 1) - 1 + 1)
			} else {
				i = 1 + p - bbits
			}
			/* i is 1 plus the number of trailing zero bits in d's significand. Thus,
			   (2^m2 * 5^m5) / (2^(s2+i) * 5^s5) = (1/2 lsb of d)/10^k. */
		} else {
			j = ilim - 1
			if m5 >= j {
				m5 -= j
			} else {
				j -= m5
				s5 += j
				b5 += j
				m5 = 0
			}
			i = ilim
			if i < 0 {
				m2 -= i
				i = 0
			}
			/* (2^m2 * 5^m5) / (2^(s2+i) * 5^s5) = (1/2 * 10^(1-ilim))/10^k. */
		}
		b2 += i
		s2 += i
		mhi = big.NewInt(1)
		/* (mhi * 2^m2 * 5^m5) / (2^s2 * 5^s5) = one-half of last printed (when mode >= 2) or
		   input (when mode < 2) significant digit, divided by 10^k. */
	}

	/* We still have d/10^k = (b * 2^b2 * 5^b5) / (2^s2 * 5^s5).  Reduce common factors in
	   b2, m2, and s2 without changing the equalities. */
	if m2 > 0 && s2 > 0 {
		if m2 < s2 {
			i = m2
		} else {
			i = s2
		}
		b2 -= i
		m2 -= i
		s2 -= i
	}

	/* Fold b5 into b and m5 into mhi. */
	if b5 > 0 {
		if leftright {
			if m5 > 0 {
				pow5mult(mhi, m5)
				b.Mul(mhi, b)
			}
			j = b5 - m5
			if j != 0 {
				pow5mult(b, j)
			}
		} else {
			pow5mult(b, b5)
		}
	}
	/* Now we have d/10^k = (b * 2^b2) / (2^s2 * 5^s5) and
	   (mhi * 2^m2) / (2^s2 * 5^s5) = one-half of last printed or input significant digit, divided by 10^k. */

	S := big.NewInt(1)
	if s5 > 0 {
		pow5mult(S, s5)
	}
	/* Now we have d/10^k = (b * 2^b2) / (S * 2^s2) and
	   (mhi * 2^m2) / (S * 2^s2) = one-half of last printed or input significant digit, divided by 10^k. */

	/* Check for special case that d is a normalized power of 2. */
	spec_case := false
	if mode < 2 {
		if (_word1(d) == 0) && ((_word0(d) & bndry_mask) == 0) &&
			((_word0(d) & (exp_mask & (exp_mask << 1))) != 0) {
			/* The special case.  Here we want to be within a quarter of the last input
			   significant digit instead of one half of it when the decimal output string's value is less than d.  */
			b2 += log2P
			s2 += log2P
			spec_case = true
		}
	}

	/* Arrange for convenient computation of quotients:
	 * shift left if necessary so divisor has 4 leading 0 bits.
	 *
	 * Perhaps we should just compute leading 28 bits of S once
	 * and for all and pass them and a shift to quorem, so it
	 * can do shifts and ors to compute the numerator for q.
	 */
	S_bytes := S.Bytes()
	var S_hiWord uint32
	for idx := 0; idx < 4; idx++ {
		S_hiWord = S_hiWord << 8
		if idx < len(S_bytes) {
			S_hiWord |= uint32(S_bytes[idx])
		}
	}
	var zz int
	if s5 != 0 {
		zz = 32 - hi0bits(S_hiWord)
	} else {
		zz = 1
	}
	i = (zz + s2) & 0x1f
	if i != 0 {
		i = 32 - i
	}
	/* i is the number of leading zero bits in the most significant word of S*2^s2. */
	if i > 4 {
		i -= 4
		b2 += i
		m2 += i
		s2 += i
	} else if i < 4 {
		i += 28
		b2 += i
		m2 += i
		s2 += i
	}
	/* Now S*2^s2 has exactly four leading zero bits in its most significant word. */
	if b2 > 0 {
		b = b.Lsh(b, uint(b2))
	}
	if s2 > 0 {
		S.Lsh(S, uint(s2))
	}
	/* Now we have d/10^k = b/S and
	   (mhi * 2^m2) / S = maximum acceptable error, divided by 10^k. */
	if k_check {
		if b.Cmp(S) < 0 {
			k--
			b.Mul(b, big10) /* we botched the k estimate */
			if leftright {
				mhi.Mul(mhi, big10)
			}
			ilim = ilim1
		}
	}
	/* At this point 1 <= d/10^k = b/S < 10. */

	if ilim <= 0 && mode > 2 {
		/* We're doing fixed-mode output and d is less than the minimum nonzero output in this mode.
		   Output either zero or the minimum nonzero output depending on which is closer to d. */
		if ilim >= 0 {
			i = b.Cmp(S.Mul(S, big5))
		}
		if ilim < 0 || i < 0 || i == 0 && !biasUp {
			/* Always emit at least one digit.  If the number appears to be zero
			   using the current mode, then emit one '0' digit and set decpt to 1. */
			if sign {
				buf = buf[:1]
			} else {
				buf = buf[:0]
			}
			buf = append(buf, '0')
			return buf, 1
		}
		buf = append(buf, '1')
		k++
		return buf, int(k + 1)
	}

	var dig byte
	if leftright {
		if m2 > 0 {
			mhi.Lsh(mhi, uint(m2))
		}

		/* Compute mlo -- check for special case
		 * that d is a normalized power of 2.
		 */

		mlo = mhi
		if spec_case {
			mhi = mlo
			mhi = new(big.Int).Lsh(mhi, log2P)
		}
		/* mlo/S = maximum acceptable error, divided by 10^k, if the output is less than d. */
		/* mhi/S = maximum acceptable error, divided by 10^k, if the output is greater than d. */

		for i = 1; ; i++ {
			z := new(big.Int)
			z.DivMod(b, S, b)
			dig = byte(z.Int64() + '0')
			/* Do we yet have the shortest decimal string
			 * that will round to d?
			 */
			j = b.Cmp(mlo)
			/* j is b/S compared with mlo/S. */
			delta := new(big.Int).Sub(S, mhi)
			var j1 int
			if delta.Sign() <= 0 {
				j1 = 1
			} else {
				j1 = b.Cmp(delta)
			}
			/* j1 is b/S compared with 1 - mhi/S. */
			if (j1 == 0) && (mode == 0) && ((_word1(d) & 1) == 0) {
				if dig == '9' {
					var flag bool
					buf = append(buf, '9')
					if buf, flag = roundOff(buf); flag {
						k++
						buf = append(buf, '1')
					}
					return buf, int(k + 1)
				}
				if j > 0 {
					dig++
				}
				buf = append(buf, dig)
				return buf, int(k + 1)
			}
			if (j < 0) || ((j == 0) && (mode == 0) && ((_word1(d) & 1) == 0)) {
				if j1 > 0 {
					/* Either dig or dig+1 would work here as the least significant decimal digit.
					   Use whichever would produce a decimal value closer to d. */
					b.Lsh(b, 1)
					j1 = b.Cmp(S)
					if (j1 > 0) || (j1 == 0 && (((dig & 1) == 1) || biasUp)) {
						dig++
						if dig == '9' {
							buf = append(buf, '9')
							buf, flag := roundOff(buf)
							if flag {
								k++
								buf = append(buf, '1')
							}
							return buf, int(k + 1)
						}
					}
				}
				buf = append(buf, dig)
				return buf, int(k + 1)
			}
			if j1 > 0 {
				if dig == '9' { /* possible if i == 1 */
					//                    round_9_up:
					//                        *s++ = '9';
					//                        goto roundoff;
					buf = append(buf, '9')
					buf, flag := roundOff(buf)
					if flag {
						k++
						buf = append(buf, '1')
					}
					return buf, int(k + 1)
				}
				buf = append(buf, dig+1)
				return buf, int(k + 1)
			}
			buf = append(buf, dig)
			if i == ilim {
				break
			}
			b.Mul(b, big10)
			if mlo == mhi {
				mhi.Mul(mhi, big10)
			} else {
				mlo.Mul(mlo, big10)
				mhi.Mul(mhi, big10)
			}
		}
	} else {
		for i = 1; ; i++ {
			//                (char)(dig = quorem(b,S) + '0');
			z := new(big.Int)
			z.DivMod(b, S, b)
			dig = byte(z.Int64() + '0')
			buf = append(buf, dig)
			if i >= ilim {
				break
			}

			b.Mul(b, big10)
		}
	}
	/* Round off last digit */

	b.Lsh(b, 1)
	j = b.Cmp(S)
	if (j > 0) || (j == 0 && (((dig & 1) == 1) || biasUp)) {
		var flag bool
		buf, flag = roundOff(buf)
		if flag {
			k++
			buf = append(buf, '1')
			return buf, int(k + 1)
		}
	} else {
		buf = stripTrailingZeroes(buf)
	}

	return buf, int(k + 1)
}

func insert(b []byte, p int, c byte) []byte {
	b = append(b, 0)
	copy(b[p+1:], b[p:])
	b[p] = c
	return b
}

func FToStr(d float64, mode FToStrMode, precision int, buffer []byte) []byte {
	if mode == ModeFixed && (d >= 1e21 || d <= -1e21) {
		mode = ModeStandard
	}

	buffer, decPt := ftoa(d, dtoaModes[mode], mode >= ModeFixed, precision, buffer)
	if decPt != 9999 {
		exponentialNotation := false
		minNDigits := 0 /* Minimum number of significand digits required by mode and precision */
		sign := false
		nDigits := len(buffer)
		if len(buffer) > 0 && buffer[0] == '-' {
			sign = true
			nDigits--
		}

		switch mode {
		case ModeStandard:
			if decPt < -5 || decPt > 21 {
				exponentialNotation = true
			} else {
				minNDigits = decPt
			}
		case ModeFixed:
			if precision >= 0 {
				minNDigits = decPt + precision
			} else {
				minNDigits = decPt
			}
		case ModeExponential:
			//                    JS_ASSERT(precision > 0);
			minNDigits = precision
			fallthrough
		case ModeStandardExponential:
			exponentialNotation = true
		case ModePrecision:
			//                    JS_ASSERT(precision > 0);
			minNDigits = precision
			if decPt < -5 || decPt > precision {
				exponentialNotation = true
			}
		}

		for nDigits < minNDigits {
			buffer = append(buffer, '0')
			nDigits++
		}

		if exponentialNotation {
			/* Insert a decimal point if more than one significand digit */
			if nDigits != 1 {
				p := 1
				if sign {
					p = 2
				}
				buffer = insert(buffer, p, '.')
			}
			buffer = append(buffer, 'e')
			if decPt-1 >= 0 {
				buffer = append(buffer, '+')
			}
			buffer = strconv.AppendInt(buffer, int64(decPt-1), 10)
		} else if decPt != nDigits {
			/* Some kind of a fraction in fixed notation */
			//                JS_ASSERT(decPt <= nDigits);
			if decPt > 0 {
				/* dd...dd . dd...dd */
				if sign {
					buffer = insert(buffer, decPt+1, '.')
				} else {
					buffer = insert(buffer, decPt, '.')
				}
			} else {
				/* 0 . 00...00dd...dd */
				o := 0
				if sign {
					o = 1
				}
				for i := 0; i < 1-decPt; i++ {
					buffer = insert(buffer, o, '0')
				}
				buffer = insert(buffer, o+1, '.')
			}
		}
	}
	return buffer
}

func bumpUp(buf []byte, k int) ([]byte, int) {
	var lastCh byte
	stop := 0
	if len(buf) > 0 && buf[0] == '-' {
		stop = 1
	}
	for {
		lastCh = buf[len(buf)-1]
		buf = buf[:len(buf)-1]
		if lastCh != '9' {
			break
		}
		if len(buf) == stop {
			k++
			lastCh = '0'
			break
		}
	}
	buf = append(buf, lastCh+1)
	return buf, k
}

func setWord0(d float64, w uint32) float64 {
	dBits := math.Float64bits(d)
	return math.Float64frombits(uint64(w)<<32 | dBits&0xffffffff)
}

func _word0(d float64) uint32 {
	dBits := math.Float64bits(d)
	return uint32(dBits >> 32)
}

func _word1(d float64) uint32 {
	dBits := math.Float64bits(d)
	return uint32(dBits)
}

func stripTrailingZeroes(buf []byte) []byte {
	bl := len(buf) - 1
	for bl >= 0 && buf[bl] == '0' {
		bl--
	}
	return buf[:bl+1]
}

/* Set b = b * 5^k.  k must be nonnegative. */
// XXXX the C version built a cache of these
func pow5mult(b *big.Int, k int) *big.Int {
	return b.Mul(b, new(big.Int).Exp(big5, big.NewInt(int64(k)), nil))
}

func roundOff(buf []byte) ([]byte, bool) {
	i := len(buf)
	stop := 0
	if i > 0 && buf[0] == '-' {
		stop = 1
	}
	for i != stop {
		i--
		if buf[i] != '9' {
			buf[i]++
			return buf[:i+1], false
		}
	}
	return buf[:stop], true
}
