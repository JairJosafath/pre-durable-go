var C1=48,C2=606,C3=1204,W1=400,W2=440,W3=400,C0=96,CH=240,JS='{"ID":"A-17","Total":4200}';
var V1=['Order{','  ID:     "A-17",','  Total:  4200,','  secret: "tok_9f",','}'],V2=['Order{','  ID:     "A-17",','  Total:  4200,','  secret: "",','}'];
function card(X,CW,title,a){if(!(a>0))return;g.globalAlpha=a;rr(X,C0,CW,CH,16);g.fillStyle='#16222E';g.fill();g.strokeStyle=LN;g.lineWidth=3;g.stroke();g.globalAlpha=1;txt(title,X,C0-30,'600 24px '+SA,MU,a)}
function arrow(x1,x2,k,label){if(!(k>0))return;var y=C0+CH/2,xe=mix(x1,x2,eo(k));seg(x1,y,xe,y,CY,4,1);if(k>=1){seg(x2-14,y-11,x2,y,CY,4,1);seg(x2-14,y+11,x2,y,CY,4,1)}txt(label,(x1+x2)/2,y-28,'500 22px '+MO,CY,k,'center')}
function draw(t){
var i,a1=P(t,.3,.7),a3=P(t,4.6,5);
card(C1,W1,'GO VALUE',a1);
for(i=0;5>i;i++){txt(V1[i],C1+28,C0+44+i*38,'500 24px '+MO,i===3?(t>=2.9?PK:TX):TX,a1)}
arrow(C1+W1+14,C2-14,P(t,1.3,1.7),'Marshal');
card(C2,W2,'CHECKPOINT',P(t,1.5,1.9));
txt(JS.slice(0,Math.floor(P(t,1.9,3)*JS.length)),C2+28,C0+CH/2,'500 24px '+MO,TX,1);
txt('unexported: not serialized',C1,C0+CH+36,'600 24px '+SA,PK,P(t,2.9,3.3));
arrow(C2+W2+14,C3-14,P(t,4.2,4.6),'Unmarshal');
card(C3,W3,'ON REPLAY',a3);
for(i=0;5>i;i++){txt(V2[i],C3+28,C0+44+i*38,'500 24px '+MO,i===3?PK:TX,P(t,4.9+i*.18,5.2+i*.18))}
txt('lost',C3+W3-28,C0+44+3*38,'600 24px '+SA,PK,P(t,6,6.4),'right');
txt('A step result is marshaled into its checkpoint.',48,468,'500 30px '+SA,TX,win(t,.5,4.2));
txt('The first run takes the same round trip, so it sees what replay will see.',48,468,'500 30px '+SA,TX,P(t,4.2,4.6));
}
