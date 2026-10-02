package capturequalification

import "strings"

const SyntheticRegistrationForm = `<!doctype html><html><body><form method="post" action="/registration-complete">
<div id="identity"><label>Email<input type="email" role="textbox" name="email" required></label><label>Password<input type="password" role="textbox" name="password" required></label>
<label>Contact name<input name="contact" required></label>
<label for="kind">Account kind</label><select id="kind" name="kind" required onchange="document.getElementById('company').hidden=this.value!=='business'"><option value="individual">Individual</option><option value="business">Business</option></select>
<div id="company" hidden><label>Company name<input name="company"></label></div>
<button type="button" onclick="document.getElementById('identity').hidden=true;document.getElementById('contact').hidden=false">Next</button></div>
<div id="contact" hidden><label>Phone<input type="tel" name="phone"></label><label>Product updates<input type="checkbox" name="updates"></label><label>Quantity<input type="text" inputmode="numeric" name="quantity" required></label><label>Ratio<input type="text" inputmode="decimal" name="ratio" required></label><button type="submit">Register</button></div></form></body></html>`

var SyntheticVerificationRegistrationForm = strings.Replace(SyntheticRegistrationForm, "</form>", `<input type="hidden" name="action" value="synthetic-named-control-canary"><input type="hidden" name="method" value="synthetic-named-control-canary"><input type="hidden" name="target" value="synthetic-named-control-canary"><input type="hidden" name="contains" value="synthetic-named-control-canary"><input type="hidden" name="append" value="synthetic-named-control-canary"><input type="hidden" name="submit" value="synthetic-named-control-canary"><div class="cf-turnstile"></div><input type="hidden" name="cf-turnstile-response" value="synthetic-verification-canary"></form><script>window.turnstile={getResponse:()=>"synthetic-verification-canary",isExpired:()=>false};</script>`, 1)
