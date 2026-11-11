var SX=450,B0=640,B1=920,IX=1030,CX=1160,YA=160,YB=400;
function base(Y,title,inside){txt(title,48,Y,'600 26px '+MO,TX,1);seg(SX,Y,1250,Y,LN,4,1);rr(B0,Y-44,B1-B0,88,16);g.fillStyle=inside?'#1F3A48':'#16222E';g.fill();g.strokeStyle=inside?CY:LN;g.lineWidth=4;g.stroke();txt('step body',(B0+B1)/2,Y-70,'500 24px '+SA,MU,1,'center');txt('checkpoint',CX,Y+56,'500 24px '+SA,MU,1,'center')}
function node(Y,done){disc(CX,Y,24,BG,1);ring(CX,Y,24,done?OR:LN,4,1);if(done)disc(CX,Y,15,OR,1)}
function zap(Y,a){if(!(a>0))return;seg(IX-16,Y-16,IX+16,Y+16,PK,6,a);seg(IX-16,Y+16,IX+16,Y-16,PK,6,a);txt('interrupted',IX,Y-70,'600 24px '+SA,PK,a,'center')}
function draw(t){
var x1=mix(mix(SX,B1,eio(P(t,.6,1.9))),IX,eio(P(t,1.9,2.3))),a1=P(t,.5,.7)*(1-P(t,2.3,2.5)),x2=mix(mix(SX,B1,eio(P(t,3.1,4.4))),CX,eio(P(t,4.4,5.2))),a2=P(t,3,3.2)*(1-P(t,5.2,5.4));
var inA=(a1>0?(x1>B0?(B1>x1?1:0):0):0)+(a2>0?(x2>B0?(B1>x2?1:0):0):0);
base(YA,'AtLeastOncePerRetry',inA>0);node(YA,t>=5.2);zap(YA,P(t,2.3,2.5));
disc(x1,YA,15,CY,a1);disc(x2,YA,15,CY,a2);txt('replay',SX,YA-40,'600 24px '+SA,OR,win(t,3,4.4));
var nA=t>=4.4?2:(t>=1.9?1:0);txt('charges: '+nA,1420,YA,'600 28px '+MO,nA>1?PK:TX,P(t,1.9,2.1));txt('ran twice',1420,YA+42,'600 24px '+SA,PK,P(t,4.4,4.8));
txt('The body can run twice: make the step idempotent.',SX,YA+104,'500 24px '+SA,TX,P(t,5.2,5.6));
var y1=mix(mix(SX,B1,eio(P(t,6.4,7.7))),IX,eio(P(t,7.7,8.1))),b1=P(t,6.3,6.5)*(1-P(t,8.1,8.3)),y2=mix(SX,B0-22,eio(P(t,8.9,9.6))),b2=P(t,8.8,9);
var inB=b1>0?(y1>B0?(B1>y1?1:0):0):0;
base(YB,'AtMostOncePerRetry',inB>0);node(YB,0);zap(YB,P(t,8.1,8.3));
disc(y1,YB,15,CY,b1);disc(y2,YB,15,MU,b2);txt('replay',SX,YB-40,'600 24px '+SA,OR,win(t,8.8,9.8));
txt('not run again',(B0+B1)/2,YB,'600 24px '+SA,MU,P(t,9.6,9.9),'center');
txt('charges: 1',1420,YB,'600 28px '+MO,TX,P(t,7.7,7.9));txt('or was it?',1420,YB+42,'600 24px '+SA,PK,P(t,9.9,10.3));
var e=P(t,9.9,10.3);if(e>0){g.globalAlpha=e;rr(1066,YB-27,316,54,12);g.fillStyle=PK;g.fill();g.globalAlpha=1;txt('StepInterruptedError',1224,YB+1,'600 24px '+MO,BG,e,'center')}
txt('The body is never run again: check the external system.',SX,YB+104,'500 24px '+SA,TX,P(t,10.2,10.6));
}
