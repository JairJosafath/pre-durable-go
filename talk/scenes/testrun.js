function block(X,w,k,label){txt(label,X,112,'600 24px '+SA,MU,P(k,0,.3));if(!(k>0))return;rr(X,140,Math.max(28,w*eo(k)),60,14);g.fillStyle=CY;g.fill()}
function draw(t){
var i,ff=win(t,1.5,2.5),k=P(t,1.5,2.3);
txt('go test',40,48,'600 30px '+MO,CY,1);txt('durabletest.NewLocalRunner',180,48,'500 24px '+MO,MU,1);
seg(40,170,780,170,LN,4,1);
block(40,200,P(t,.5,1.3),'invocation 1');
seg(252,170,mix(252,558,k),170,OR,6,k>0?1:0,1);
txt('Wait: 1 hour',405,112,'600 24px '+SA,OR,P(t,1.2,1.5),'center');
txt(Math.round(60*k)+' min',405,214,'600 28px '+MO,OR,P(t,1.5,1.7),'center');
txt('completed by the runner',405,252,'500 24px '+SA,MU,P(t,2.3,2.6),'center');
for(i=0;3>i;i++){var x=262+((t*420+i*96)%280);seg(x,158,x+12,170,OR,4,ff);seg(x,182,x+12,170,OR,4,ff)}
block(570,210,P(t,2.5,3.3),'invocation 2');
var s=P(t,3.5,3.9);if(s>0){g.save();g.translate(40,316);g.scale(bk(s),bk(s));txt('SUCCEEDED',0,0,'700 40px '+SA,CY,s);g.restore()}
txt('got 42',330,318,'500 28px '+MO,TX,P(t,3.8,4.1));
txt('no network',40,384,'600 24px '+SA,MU,P(t,4.3,4.6));txt('no AWS credentials',210,384,'600 24px '+SA,MU,P(t,4.6,4.9));txt('service limits enforced',478,384,'600 24px '+SA,MU,P(t,4.9,5.2));
}
