var LX=420,RX=1580,B0=820,B1=1100,YA=170,YB=410;
function lane(Y,title,sub,sa){txt(title,48,Y-16,'600 30px '+SA,TX,1);txt(sub,48,Y+24,'500 24px '+SA,MU,sa);seg(LX,Y,RX,Y,LN,4,1)}
function box(Y,col){rr(B0,Y-46,B1-B0,92,16);g.fillStyle='#16222E';g.fill();g.strokeStyle=col;g.lineWidth=4;g.stroke();txt('durable.Step',(B0+B1)/2,Y-72,'500 24px '+MO,MU,1,'center')}
function spark(X,Y,k){if(!(k>0))return;if(k>=1)return;var i;for(i=0;8>i;i++){var a=i*.785;seg(X+Math.cos(a)*(12+k*30),Y+Math.sin(a)*(12+k*30),X+Math.cos(a)*(24+k*48),Y+Math.sin(a)*(24+k*48),PK,4,1-k)}}
function draw(t){
var xa=mix(mix(mix(mix(mix(mix(LX,950,eio(P(t,.6,1.6))),B0+24,eio(P(t,2,2.4))),1010,eio(P(t,2.8,3.5))),B0+24,eio(P(t,3.9,4.3))),B1,eio(P(t,4.7,5.5))),RX,eio(P(t,5.5,6.3)));
lane(YA,'Inside a step','retried by the step strategy',P(t,2,2.4));
box(YA,t>=5.5?CY:(t>=1.6?OR:LN));
spark(950,YA,P(t,1.6,2.1));spark(1010,YA,P(t,3.5,4));
txt('attempt 1 failed',1130,YA-72,'600 24px '+SA,PK,win(t,1.6,2.8));txt('attempt 2 failed',1130,YA-72,'600 24px '+SA,PK,win(t,3.5,4.7));txt('attempt 3 succeeded',1130,YA-72,'600 24px '+SA,CY,P(t,5.5,5.8));
seg(LX,YA,xa,YA,CY,4,P(t,.6,.8));disc(xa,YA,15,CY,P(t,.5,.7));
txt('the handler continues',RX,YA+44,'600 24px '+SA,CY,P(t,6.1,6.5),'right');
var xb=mix(LX,1330,eio(P(t,7,8.4))),bad=P(t,8.4,8.7);
lane(YB,'Outside a step','no retry: the execution fails',P(t,8.6,9));
box(YB,xb>B1?CY:LN);
seg(LX,YB,xb,YB,bad>.5?PK:CY,4,P(t,7,7.2));disc(xb,YB,15,bad>.5?PK:CY,P(t,6.9,7.1));
spark(1330,YB,P(t,8.4,8.9));
txt('error here',1330,YB-52,'600 24px '+SA,PK,P(t,8.4,8.7),'center');
var f=P(t,8.7,9.1);if(f>0){g.save();g.translate(1480,YB+2);g.scale(bk(f),bk(f));txt('FAILED',0,0,'700 44px '+SA,PK,f,'center');g.restore()}
}
