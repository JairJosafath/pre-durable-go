function draw(t){var k=eo(P(t,.25,.9)),r=P(t,.8,1.2),p=P(t,.9,1.9),TE='#006E8A';
seg(16,100,mix(16,430,k),100,TE,4,k>0?1:0);
if(r>0){g.save();g.translate(488,100);g.rotate(t*.5);g.setLineDash([10,9]);g.globalAlpha=r;g.beginPath();g.arc(0,0,48*bk(r),0,6.2832);g.strokeStyle=TE;g.lineWidth=4;g.stroke();g.setLineDash([]);g.restore();g.globalAlpha=1}
if(p>0){if(1>p){ring(488,100,48+p*30,TE,3,1-p)}}
}
